FROM alpine:3.5

MAINTAINER Xuehao Zhou <robertzhouxh@gmail.com>

RUN echo "https://mirrors.ustc.edu.cn/alpine/v3.5/main" > /etc/apk/repositories \
    && echo "https://mirrors.ustc.edu.cn/alpine/v3.5/community" >> /etc/apk/repositories \
    && apk update \
    && apk add --no-cache ca-certificates
# Build the app and copy the bin app to the container
# CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/release/app ./main.go

COPY ./message-example.conf /message.conf
COPY ./log.xml /message_log.xml
COPY ./message /

RUN set -ex \
  && mkdir -p /opt/message \
  && mv /message.conf /opt/message/message.conf \
  && mv /message_log.xml /opt/message/message_log.xml \
  && mv /message /opt/message/message

WORKDIR /opt/message

CMD ["/opt/message/message", "-c", "/opt/message/message.conf"]

EXPOSE 8070 8270 8170
