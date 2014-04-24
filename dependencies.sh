#!/bin/bash
# Dependencies

# go get -u github.com/Terry-Mao/gopush-cluster
# go get -u github.com/Terry-Mao/goconf
# go get -u github.com/garyburd/redigo/redis
# go get -u code.google.com/p/go.net/websocket
# go get -u github.com/samuel/go-zookeeper
# go get -u github.com/golang/glog

go get -u ./message/...
go get -u ./comet/...
go get -u ./web/...
