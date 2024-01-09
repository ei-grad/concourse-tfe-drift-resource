FROM golang:alpine as build
WORKDIR /go/src/concourse-tfe-drift-resource
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w"

FROM alpine
COPY --from=build /go/src/concourse-tfe-drift-resource/concourse-tfe-drift-resource /opt/resource/concourse-tfe-drift-resource
RUN for i in "check" "in" "out"; do \
        ln -s /opt/resource/concourse-tfe-drift-resource /opt/resource/$i; \
    done
