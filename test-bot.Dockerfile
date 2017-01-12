FROM msoap/shell2telegram

RUN apk add --no-cache ca-certificates
ENV TB_TOKEN=*******
CMD ["/date", "date"]
