FROM composer as composer

FROM php:7.4-buster as grpc-base

RUN apt-get -qq update && apt-get -qq install -y \
  autoconf automake curl git libtool \
  pkg-config unzip zlib1g-dev

ARG MAKEFLAGS=-j8

WORKDIR /tmp

RUN curl -sSL https://github.com/protocolbuffers/protobuf/releases/download/v3.14.0/\
protoc-3.14.0-linux-x86_64.zip -o /tmp/protoc.zip && \
  unzip -qq protoc.zip && \
  cp /tmp/bin/protoc /usr/local/bin/protoc && \
  cp -R /tmp/include /usr/local/


WORKDIR /github/grpc

RUN git clone https://github.com/grpc/grpc . && \
  git submodule update --init && \
  cd third_party/protobuf && git submodule update --init

RUN make grpc_php_plugin

RUN pecl install grpc
RUN pecl install protobuf

FROM debian:buster as sprial

RUN apt-get -qq update && apt-get -qq install -y curl wget git

RUN wget https://github.com/spiral/php-grpc/releases/download/v1.4.1/protoc-gen-php-grpc-1.4.1-linux-amd64.tar.gz -O protoc-gen-php-grpc-1.4.1-linux-amd64.tar.gz && \
    wget https://github.com/spiral/php-grpc/releases/download/v1.4.1/rr-grpc-1.4.1-linux-amd64.tar.gz -O rr-grpc-1.4.1-linux-amd64.tar.gz && \
    tar xvfz protoc-gen-php-grpc-1.4.1-linux-amd64.tar.gz && \
    tar xvfz rr-grpc-1.4.1-linux-amd64.tar.gz && \
    cp protoc-gen-php-grpc-1.4.1-linux-amd64/protoc-gen-php-grpc /usr/local/bin/ && \
    cp rr-grpc-1.4.1-linux-amd64/rr-grpc /usr/local/bin/

FROM php:7.4-buster

RUN apt-get -qq update && apt-get -qq install -y git unzip libzip-dev

COPY --from=composer /usr/bin/composer /usr/bin/composer

COPY --from=grpc-base /usr/local/bin/protoc /usr/local/bin/protoc

COPY --from=grpc-base /usr/local/include/google /usr/local/include/google

COPY --from=grpc-base /github/grpc/bins/opt/grpc_php_plugin /usr/local/bin/protoc-gen-grpc

COPY --from=grpc-base \
  /usr/local/lib/php/extensions/no-debug-non-zts-20190902/grpc.so \
  /usr/local/lib/php/extensions/no-debug-non-zts-20190902/grpc.so

COPY --from=grpc-base \
  /usr/local/lib/php/extensions/no-debug-non-zts-20190902/protobuf.so \
  /usr/local/lib/php/extensions/no-debug-non-zts-20190902/protobuf.so

COPY --from=sprial /usr/local/bin/protoc-gen-php-grpc /usr/local/bin/protoc-gen-php-grpc
COPY --from=sprial /usr/local/bin/rr-grpc /usr/local/bin/rr-grpc

RUN docker-php-ext-install zip
RUN docker-php-ext-enable grpc protobuf

ENV KUBECTL_VERSION=1.19.0
RUN cd /tmp && curl -sLf https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl \
    && chmod +x /usr/local/bin/kubectl \
    && rm -rf /tmp/*

WORKDIR /var/www/html

COPY composer.json .
RUN composer install

COPY ./protos/externalscaler.proto .
RUN protoc --proto_path=./ --php_out=. --grpc_out=. --plugin=protoc-gen-grpc=/usr/local/bin/protoc-gen-php-grpc ./externalscaler.proto

COPY src/.rr.yaml ./
COPY src/worker.php .
COPY src/ExternalScaler.php .

RUN chmod 644 worker.php

EXPOSE 9001

CMD ["rr-grpc", "serve", "-v", "-d"]
#docker run -it --rm --net=host -v $(pwd):/app -w /app -p 8080:8080 fullstorydev/grpcui -plaintext -proto /app/protos/externalscaler.proto -import-path /app/protos/  localhost:9001

