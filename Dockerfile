# build image
FROM golang:alpine as go_builder

RUN apk add --no-cache git

RUN go get -v github.com/msoap/shell2telegram
RUN cd /go/src/github.com/msoap/shell2telegram && go install -a -v -ldflags="-w -s" ./...

# final image
FROM alpine

RUN apk add --no-cache ca-certificates
COPY --from=go_builder /go/bin/shell2telegram /app/shell2telegram
ENTRYPOINT ["/app/shell2telegram"]
CMD ["-help"]
