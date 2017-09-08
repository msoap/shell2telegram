# build image
FROM golang:alpine as go_builder

RUN apk add --no-cache git

ENV CGO_ENABLED=0
RUN go get -a -v -ldflags="-w -s" github.com/msoap/shell2telegram

# final image
FROM alpine

RUN apk add --no-cache ca-certificates
COPY --from=go_builder /go/bin/shell2telegram /app/shell2telegram
ENTRYPOINT ["/app/shell2telegram"]
CMD ["-help"]
