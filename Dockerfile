FROM golang:1.19-alpine3.16

# Upgrade and Install bash
RUN apk update && apk upgrade \
    && apk add --no-cache bash

# Set the Current Working Directory inside the container
WORKDIR /go/src/app

# Copy sources
COPY go.mod go.sum wordpress_exporter.go ./

# Download all the dependencies
RUN go get -d -v ./...

# Install the package
RUN go install -v ./...

# Default env
ENV WORDPRESS_DB_HOST="" \
    WORDPRESS_DB_PORT="3306" \
    WORDPRESS_DB_USER="" \
    WORDPRESS_DB_PASSWORD="" \
    WORDPRESS_DB_NAME="" \
    WORDPRESS_TABLE_PREFIX="wp_" \
    WORDPRESS_SKIP_WOOCOMMERCE="false"

EXPOSE 9850

ADD /docker-entrypoint.sh /docker-entrypoint.sh

RUN set -x \
  && chmod +x /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]

CMD ["wordpress_exporter"]
