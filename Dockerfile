# build stage
FROM golang:alpine as build-env
MAINTAINER maintainer="mdouchement"

RUN apk upgrade && apk add curl git ca-certificates
RUN update-ca-certificates

RUN curl -sL https://taskfile.dev/install.sh | sh

ENV CGO_ENABLED 0
ENV GO111MODULE on

WORKDIR /geoblock-proxy
COPY . .

RUN go mod download
RUN task build-server

# final stage
FROM scratch
MAINTAINER maintainer="mdouchement"

ENV GEOBLOCK_PROXY_CONFIG /etc/geoblock-proxy/geoblock-proxy.yml

COPY --from=build-env /geoblock-proxy/dist/geoblock-proxy-linux-amd64 /usr/local/bin/geoblock-proxy
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080
CMD ["geoblock-proxy"]
