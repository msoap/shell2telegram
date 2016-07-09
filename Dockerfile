FROM alpine

ADD shell2telegram /app/shell2telegram
ENV PATH=$PATH:/app
ENTRYPOINT ["/app/shell2telegram"]
CMD ["-help"]
