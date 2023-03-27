package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// PatchRecord represents a single patch for modifying a resource.
type patchRecord struct {
	Op    string      `json:"op,inline"`
	Path  string      `json:"path,inline"`
	Value interface{} `json:"value"`
}

type handler struct {
}

func newHandler() *handler {
	h := &handler{}
	return h
}

func (h *handler) initResourchesPatch() patchRecord {
	return patchRecord{
		Op:    "add",
		Path:  "/spec/containers/0/resources",
		Value: corev1.ResourceRequirements{},
	}
}

func (h *handler) initResourceValue(kind string) patchRecord {
	return patchRecord{
		Op:    "add",
		Path:  fmt.Sprintf("/spec/containers/0/resources/%s", kind),
		Value: corev1.ResourceList{},
	}
}

func (h *handler) initRequirementValuePatch(kind string, requirement string, quantity string) patchRecord {
	return patchRecord{
		Op:    "add",
		Path:  fmt.Sprintf("/spec/containers/0/resources/%s/%s", kind, requirement),
		Value: quantity}
}

func (h *handler) getPatches(ar v1.AdmissionRequest) ([]patchRecord, error) {
	if ar.Resource.Version != "v1" {
		return nil, fmt.Errorf("only v1 Pods are supported")
	}

	raw, namespace := ar.Object.Raw, ar.Namespace
	pod := corev1.Pod{}
	if err := json.Unmarshal(raw, &pod); err != nil {
		return nil, err
	}
	if len(pod.Name) == 0 {
		pod.Name = pod.GenerateName + "%"
		pod.Namespace = namespace
	}

	mark := pod.Spec.Containers[0].Resources

	patches := []patchRecord{}

	if mark.Limits == nil && mark.Requests == nil {
		patches = append(patches, h.initResourchesPatch())
	}

	if mark.Limits == nil {
		patches = append(patches, h.initResourceValue("limits"))
	}

	if mark.Requests == nil {
		patches = append(patches, h.initResourceValue("requests"))
	}

	if mark.Limits == nil || mark.Limits.Cpu().String() == "" {
		patches = append(patches, h.initRequirementValuePatch("limits", "cpu", "200m"))
	}

	if mark.Limits == nil || mark.Limits.Memory().String() == "" {
		patches = append(patches, h.initRequirementValuePatch("limits", "memory", "2048Ki"))
	}

	if mark.Requests == nil || mark.Requests.Cpu().String() == "" {
		patches = append(patches, h.initRequirementValuePatch("requests", "cpu", "100m"))
	}

	if mark.Requests == nil || mark.Requests.Memory().String() == "" {
		patches = append(patches, h.initRequirementValuePatch("requests", "memory", "1024Ki"))
	}

	return patches, nil
}

func (h *handler) admit(data []byte) *v1.AdmissionResponse {
	response := v1.AdmissionResponse{}
	response.Allowed = true

	review := v1.AdmissionReview{}
	if err := json.Unmarshal(data, &review); err != nil {
		klog.Error(err)
		return &response
	}

	response.UID = review.Request.UID

	patches, err := h.getPatches(*review.Request)

	if err != nil {
		klog.Error(err)
		return &response
	}

	if len(patches) > 0 {
		patch, err := json.Marshal(patches)
		if err != nil {
			klog.Errorf("Cannot marshal the patch %v: %v", patches, err)
			return &response
		}
		patchType := v1.PatchTypeJSONPatch
		response.PatchType = &patchType
		response.Patch = patch
		klog.V(4).Infof("Sending patches: %v", patches)
	}

	return &response
}

func (h *handler) Serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	reviewResponse := h.admit(body)
	ar := v1.AdmissionReview{
		Response: reviewResponse,
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
	}

	resp, err := json.Marshal(ar)
	if err != nil {
		klog.Error(err)
		return
	}

	_, err = w.Write(resp)
	if err != nil {
		klog.Error(err)
		return
	}
}
