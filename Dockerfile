FROM alpine

RUN apk add --no-cache ca-certificates
ADD shell2telegram /app/shell2telegram
ENTRYPOINT ["/app/shell2telegram"]
CMD ["-help"]
