FROM alpine:3.5

MAINTAINER Xuehao Zhou <robertzhouxh@gmail.com>


RUN echo "https://mirrors.ustc.edu.cn/alpine/v3.5/main" > /etc/apk/repositories \
    && echo "https://mirrors.ustc.edu.cn/alpine/v3.5/community" >> /etc/apk/repositories \
    && apk update \
    && apk add --no-cache ca-certificates
# Build the app and copy the bin app to the container
# CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/release/app ./main.go

COPY ./comet-example.conf /comet.conf
COPY ./log.xml /comet_log.xml
COPY ./comet /comet

RUN set -ex \
  && mkdir -p /opt/comet \
  && mv /comet.conf /opt/comet/comet.conf \
  && mv /comet_log.xml /opt/comet/comet_log.xml \
  && mv /comet /opt/comet/comet

WORKDIR /opt/comet

CMD ["/opt/comet/comet", "-c", "/opt/comet/comet.conf"]

EXPOSE 6968 7068 6969 7069 6970 7070 6071 6072


