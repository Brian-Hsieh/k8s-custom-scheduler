FROM alpine:latest

# Install CA certificates for HTTPS support
RUN apk --no-cache add ca-certificates

COPY ./custom-scheduler /usr/local/bin/custom-scheduler

RUN chmod +x /usr/local/bin/custom-scheduler

CMD ["/usr/local/bin/custom-scheduler"]

