# build stage
FROM golang:alpine as build-env
MAINTAINER mdouchement

RUN apk upgrade && apk add curl git
RUN curl -sL https://taskfile.dev/install.sh | sh

ENV CGO_ENABLED 0
ENV GO111MODULE on

WORKDIR /geoblock-proxy
COPY . .

RUN go mod download
RUN task build-server

# final stage
FROM alpine
MAINTAINER mdouchement

ENV GEOBLOCK_PROXY_CONFIG /etc/geoblock-proxy/geoblock-proxy.yml

RUN apk upgrade && apk add ca-certificates

COPY --from=build-env /geoblock-proxy/dist/geoblock-proxy-linux-amd64 /usr/local/bin/geoblock-proxy

EXPOSE 8080
CMD ["geoblock-proxy"]
