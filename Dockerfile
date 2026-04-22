FROM golang:1.26-alpine AS build

RUN apk --no-cache add ca-certificates
COPY . /build
RUN cd /build && go mod download && CGO_ENABLED=0 go build -a -o http-mock .

FROM scratch
WORKDIR /

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /build/http-mock /http-mock
USER 65532:65532

ENTRYPOINT ["/http-mock"]
