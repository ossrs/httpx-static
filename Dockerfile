

############################################################
# build
############################################################
FROM ossrs/srs:dev AS build

RUN yum install -y git openssl
COPY . /tmp/go-oryx
WORKDIR /tmp/go-oryx/httpx-static

# Build for alpine, see https://www.cloudbees.com/blog/building-minimal-docker-containers-for-go-applications
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo  -o objs/httpx-static .

RUN openssl genrsa -out server.key 2048 && \
    openssl req -new -x509 -key server.key -out server.crt -days 3650 \
        -subj "/C=CN/ST=Beijing/L=Beijing/O=Me/OU=Me/CN=ossrs.net" && \
    # Install binary.
    cp objs/httpx-static /usr/local/bin/httpx-static && \
    cp server.* /usr/local/etc/ && \
    cp -R html /usr/local/

############################################################
# dist
############################################################
FROM alpine:3.16 AS dist

# HTTP/80, HTTPS/443
EXPOSE 80 443
# SRS binary, config files and srs-console.
COPY --from=build /usr/local/bin/httpx-static /usr/local/bin/
COPY --from=build /usr/local/etc/server.* /usr/local/etc/
COPY --from=build /usr/local/html /usr/local/html
# Default workdir and command.
WORKDIR /usr/local
CMD ["./bin/httpx-static", \
    "-http", "80", "-https", "443", "-root", "./html", \
    "-ssk", "./etc/server.key", "-ssc", "./etc/server.crt" \
    ]
