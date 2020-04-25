# -- golang
#
FROM golang:1.14.2 as golang
MAINTAINER Matt Ho

ADD . /build
WORKDIR /build

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o fairy main.go

# -- alpine
#
FROM alpine:latest as alpine
RUN apk --no-cache add tzdata zip ca-certificates
WORKDIR /usr/share/zoneinfo
# -0 means no compression.  Needed because go's
# tz loader doesn't handle compressed data.
RUN zip -r -0 /zoneinfo.zip .

# -- image
#
FROM busybox

# cli
COPY --from=golang /build/fairy /usr/local/bin/fairy

# tz
ENV ZONEINFO /zoneinfo.zip
COPY --from=alpine /zoneinfo.zip /

# tls
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/usr/local/bin/fairy"]
