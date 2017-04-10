FROM alpine:3.5

MAINTAINER Xuehao Zhou <robertzhouxh@gmail.com>

RUN echo "https://mirrors.ustc.edu.cn/alpine/v3.5/main" > /etc/apk/repositories \
    && echo "https://mirrors.ustc.edu.cn/alpine/v3.5/community" >> /etc/apk/repositories \
    && apk update \
    && apk add --no-cache ca-certificates
# Build the app and copy the bin app to the container
# CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/release/app ./main.go

COPY ./web-example.conf /web.conf
COPY ./log.xml /web_log.xml
COPY ./web /web

RUN set -ex \
  && mkdir -p /opt/web \
  && mv /web.conf /opt/web/web.conf \
  && mv /web_log.xml /opt/web/web_log.xml \
  && mv /web /opt/web/web

WORKDIR /opt/web

CMD ["/opt/web/web", "-c", "/opt/web/web.conf"]

EXPOSE 8090 8091




