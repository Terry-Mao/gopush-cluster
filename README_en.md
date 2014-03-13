gopush-cluster
==============
`Terry-Mao/gopush-cluster` is a go push server cluster (support websocket, tcp protocol).

---------------------------------------
  * [Features](#features)
  * [Installation](#installation)
  * [Usage](#usage)
  * [Configuration](#configuration)
  * [Examples](#examples)
  * [Documentation](#documentation)
  * [FAQ](#faq)
  * [LICENSE](#license)

---------------------------------------

## Features
 * lightweight
 * high performance
 * pure golang implementation
 * message expired
 * offline message store
 * public message or private message push
 * multiple subscribers (can restrict max subscribers)
 * heartbeat（service heartbeat or tcp keepalive）
 * auth (if a subscriber not auth then can not connect to comet node)
 * multiple protocol (websocket, tcp, todo http longpolling)
 * stat
 * cluster support (easy add or remove comet & web & message node)
 * failover support (zookeeper)

## Installation
### zookeeper
1.download [zookeeper](http://www.apache.org/dyn/closer.cgi/zookeeper/),suggest version:'3.4.5'.

2.unzip package
```sh
$ mkdir -p /data/programfiles
$ cp ./zookeeper-3.4.5.tar.gz /data/programfiles
$ cd /data/programfiles/
$ tar -xvf zookeeper-3.4.5.tar.gz -C ./
```
3.compile && install
``` sh
$ cd zookeeper-3.4.5/src/c
$ ./configure
$ make && make install
```
4.start zookeeper
```sh
$ cd /data/programfiles/zookeeper-3.4.5/bin
$ nohup ./zkServer.sh start &
```
### redis
```sh
$ cd /data/programfiles
$ wget https://redis.googlecode.com/files/redis-2.6.4.tar.gz
$ tar -xvf redis-2.6.4.tar.gz -C ./
$ cd redis-2.6.4
$ make
$ make test
$ make install
$ mkdir /etc/redis
$ cp /data/programfiles/redis-2.6.4/redis.conf /etc/redis/
$ cp /data/programfiles/redis-2.6.4/redis-server /etc/init.d/redis-server
$ /etc/init.d/redis-server /etc/redis/redis.conf
```
### git
reference:[git](http://git-scm.com/download/linux)

### golang
1.download
```sh
# centos
$ wget https://go.googlecode.com/files/go1.2.linux-amd64.tar.gz
$ tar -xvf go1.2.linux-amd64.tar.gz
$ cp -R go /usr/local/
```
2.golang env

modify ~/.profile
```sh
$ vim ~/.profile
# append
export GOROOT=/usr/local/go
export PATH=$PATH:$GOROOT/bin
export GOPATH=/data/app/go
```
### gopush-cluster
1.download gopush-cluster
```sh
$ go get -u github.com/Terry-Mao/gopush-cluster
$ go get -u github.com/Terry-Mao/goconf
$ go get -u github.com/garyburd/redigo/redis
$ go get -u code.google.com/p/go.net/websocket
$ go get -u launchpad.net/gozk/zookeeper
```
*if following error, see FAQ 1

go: missing Mercurial command. See http://golang.org/s/gogetcmd

package code.google.com/p/go.net/websocket: exec: "hg": executable file not found in $PATH

*if following error, see FAQ 2

go: missing Bazaar command. See http://golang.org/s/gogetcmd

package launchpad.net/gozk/zookeeper: exec: "bzr": executable file not found in $PATH

*if following error, see FAQ 3

launchpad.net/gozk/zookeeper

../zk.go:15:23: error: zookeeper.h: No such file or directory

2.install message,comet,web node
```sh
$ cd $GOPATH/src/github.com/Terry-Mao/gopush-cluster/message
$ go install
$ cp message
$ cp message.conf $GOPATH/bin/
$ cd ../comet/
$ go install
$ cp comet-example.conf /data/app/go/bin/
$ cd ../web/
$ go install
$ cp web.conf /data/app/go/bin/
```
All done!!!

### start gopush-cluster
```sh
$ cd /$GOPATH/bin
$ nohup ./message -c message.conf &
$ nohup ./comet -c comet-example.conf &
$ nohup ./web -c web.conf &
```
*if following error, FAQ 4

error while loading shared libraries: libzookeeper_mt.so.2: cannot open shared object file: No such file or directory

### testing
1.push public message
```sh
$ curl -d "test2" http://localhost:8091/admin/push/public?expire=600
```
succeed response：{"msg":"ok","ret":0}
2.push private message
```sh
$ curl -d "test" http://localhost:8091/admin/push?key=Terry-Mao\&expire=600\&gid=0
```
succeed response：{"msg":"ok","ret":0}
3.get offline message
open http://localhost:8090/msg/get?key=Terry-Mao&mid=1&pmid=0 in browser
succeed response:
```json
{
    "data":{
        "msgs":[
            "{"msg":"test","expire":1391943609703654726,"mid":13919435497036558}"
        ],
        "pmsgs":[
            "{"msg":"test2","expire":1391943637016665915,"mid":13919435770166656}"
        ]
    },
    "msg":"ok",
    "ret":0
}
```
4.get node address
open http://localhost:8090/server/get?key=Terry-Mao&proto=2 in browser
succeed response:
```json
{
    "data":{
        "server":"localhost:6969"
    },
    "msg":"ok",
    "ret":0
}
```

## Configuration
[web](https://github.com/Terry-Mao/gopush-cluster/blob/master/web/web-example.conf)
[comet](https://github.com/Terry-Mao/gopush-cluster/blob/master/comet/comet-example.conf)
[message](https://github.com/Terry-Mao/gopush-cluster/blob/master/message/message-example.conf)

## Examples
java: [gopush-cluster-sdk](https://github.com/Terry-Mao/gopush-cluster-sdk)

ios: [GoPushforIOS](https://github.com/roy5931/GoPushforIOS)

javascript: [gopush-cluster-javascript-sdk](https://github.com/Lanfei/gopush-cluster-javascript-sdk)

## Documentation
`web`
[external](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/web/external_proto_zh.textile)
[internal](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/web/internal_proto_zh.textile)

`comet`
[client](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/comet/client_proto_zh.textile)
[internal](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/comet/rpc_proto_zh.textile)

`message`
[internal](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/message/rpc_proto_zh.textile)

## FAQ
1.install hg
```sh
$ wget http://mercurial.selenic.com/release/mercurial-1.4.1.tar.gz 
$ tar -xvf mercurial-1.4.1.tar.gz
$ cd mercurial-1.4.1
$ make
$ make install
```
2.install bzr
```sh
$ yum install bzr.x86_64
```
3.modify gozk
```sh
$ vim $GOPATH/src/launchpad.net/gozk/zookeeper/zk.go
# find this line
# cgo CFLAGS: -I/usr/include/c-client-src -I/usr/include/zookeeper
# change include path
# cgo CFLAGS: -I/usr/local/include/zookeeper
```
4.error：libzookeeper_mt.so.2 can not found
```sh
$ export LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH
$ sudo ldconfig
```

## LICENSE
TODO
