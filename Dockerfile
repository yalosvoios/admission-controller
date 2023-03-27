## Build
FROM golang:1.19-buster as build 

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
COPY *.go ./

RUN go build -o /yalos-admission-controller

## Deploy
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /yalos-admission-controller /yalos-admission-controller

USER nonroot:nonroot

ENTRYPOINT ["/yalos-admission-controller"]
