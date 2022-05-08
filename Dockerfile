# docker build -t msoap/shell2telegram .

# build image
FROM --platform=$BUILDPLATFORM golang:alpine as go_builder

RUN apk add --no-cache git

ADD . $GOPATH/src/github.com/msoap/shell2telegram
WORKDIR $GOPATH/src/github.com/msoap/shell2telegram

ENV CGO_ENABLED=0
# GOARM=6 affects only "arm" builds
ENV GOARM=6
# "amd64", "arm64" or "arm" (--platform=linux/amd64,linux/arm64,linux/arm/v6)
ENV GOARCH=$TARGETARCH
ENV GOOS=linux

RUN go build -v -trimpath -ldflags="-w -s -X 'main.version=$(git describe --abbrev=0 --tags | sed s/v//)'" -o /go/bin/shell2telegram .

# final image
FROM alpine

RUN apk add --no-cache ca-certificates
COPY --from=go_builder /go/bin/shell2telegram /app/shell2telegram
ENTRYPOINT ["/app/shell2telegram"]
CMD ["-help"]
