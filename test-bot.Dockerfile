FROM msoap/shell2telegram

# may be install some alpine packages:
# RUN apk add --no-cache ...
ENV TB_TOKEN=*******
CMD ["/date", "date"]
