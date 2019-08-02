FROM golang:1.12.7-buster

# Set the Current Working Directory inside the container
WORKDIR $GOPATH/src/wordpress_exporter

# Copy sources.
COPY . .

# Download all the dependencies.
RUN go get -d -v ./...

# Install the package
RUN go install -v ./...

ENV WORDPRESS_DB_HOST="" \
    WORDPRESS_DB_PORT="3306" \
    WORDPRESS_DB_USER="" \
    WORDPRESS_DB_PASSWORD="" \
    WORDPRESS_DB_NAME="" \
    WORDPRESS_TABLE_PREFIX="wp_"

EXPOSE 9117

ADD /docker-entrypoint.sh /docker-entrypoint.sh

RUN set -x \
  && chmod +x /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]

CMD ["wordpress_exporter"]