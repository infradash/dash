# Dash Dockerfile


# Pull base image.
FROM ubuntu:14.04

# This assumes that the binary has been copied into the directory where this Dockerfile lives.
COPY dash /usr/bin/

# SSL certs - see https://medium.com/@kelseyhightower/optimizing-docker-images-for-static-binaries-b5696e26eb07
RUN mkdir -p /etc/ssl/certs
COPY ca-certificates.crt /etc/ssl/certs/

# UI
RUN mkdir -p /var/www
COPY www /var/www/

# Expose ports.
EXPOSE 25657 25658

# Define mountable directories.
RUN mkdir -p /var/log/dash/
RUN mkdir -p /var/run/dash/

VOLUME ["/var/run/dash", "/var/log/dash"]

# Define working directory.
WORKDIR /var/run/dash

# Define default command.
ENTRYPOINT ["dash", "-logtostderr", "-ui_docroot=/var/www" ]
