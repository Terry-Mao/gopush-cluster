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
### Dependencies
```sh
$ yum -y install java-1.7.0-openjdk$ yum -y install gcc-c++
```
### zookeeper
1.mkdir
```sh
$ mkdir -p /data/apps$ mkdir -p /data/logs$ mkdir -p /data/programfiles
```
2.download [zookeeper](http://www.apache.org/dyn/closer.cgi/zookeeper/), suggest version: '3.4.5'.
```sh
$ cd /data/programfiles$ wget http://mirror.bit.edu.cn/apache/zookeeper/zookeeper-3.4.5/zookeeper-3.4.5.tar.gz$ tar -xvf zookeeper-3.4.5.tar.gz -C ./
```
3.start zookeeper
```sh
$ cp /data/programfiles/zookeeper-3.4.5/conf/zoo_sample.cfg /data/programfiles/zookeeper-3.4.5/conf/zoo.cfg
$ cd /data/programfiles/zookeeper-3.4.5/bin
$ ./zkServer.sh start
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
$ cp /data/programfiles/redis-2.6.4/src/redis-server /etc/init.d/redis-server
$ /etc/init.d/redis-server /etc/redis/redis.conf
```
* if following error, see FAQ 2
```sh
which: no tclsh8.5 in (/usr/lib64/qt-3.3/bin:/usr/local/bin:/usr/bin:/bin:/usr/local/sbin:/usr/sbin:/sbin:/home/geffzhang/bin)
You need 'tclsh8.5' in order to run the Redis test
Make[1]: *** [test] error 1
make[1]: Leaving directory ‘/data/program files/redis-2.6.4/src’
Make: *** [test] error 2！
```
### git
reference:[git](http://git-scm.com/download/linux)
```sh
$ yum -y install git
```

### golang
1.download
```sh
# centos
$ cd /data/programfiles
$ wget -c --no-check-certificate https://go.googlecode.com/files/go1.2.linux-amd64.tar.gz
$ tar -xvf go1.2.linux-amd64.tar.gz -C /usr/local
```
2.golang env
```sh
$ vim /etc/profile.d/golang.sh
# append
export GOROOT=/usr/local/go
export PATH=$PATH:$GOROOT/bin
export GOPATH=/data/apps/go
$ source /etc/profile
```
### gopush-cluster
1.download gopush-cluster
```sh
$ ./dependencies.sh
```
*if following error, see FAQ 1

go: missing Mercurial command. See http://golang.org/s/gogetcmd

package code.google.com/p/go.net/websocket: exec: "hg": executable file not found in $PATH

2.install message,comet,web node
```sh
$ cd $GOPATH/src/github.com/Terry-Mao/gopush-cluster/message
$ go install
$ cp message-example.conf $GOPATH/bin/message.conf
$ cd ../comet/
$ go install
$ cp comet-example.conf /data/apps/go/bin/comet.conf
$ cd ../web/
$ go install
$ cp web-example.conf /data/apps/go/bin/web.conf
```
All done!!!

### start gopush-cluster
```sh
$ cd /$GOPATH/bin
$ nohup ./message -c message.conf -v=1 -log_dir="/data/logs/gopush-cluster/" -stderrthreshold=FATAL &
$ nohup ./comet -c comet.conf -v=1 -log_dir="/data/logs/gopush-cluster/" -stderrthreshold=FATAL &
$ nohup ./web -c web.conf -v=1 -log_dir="/data/logs/gopush-cluster/" -stderrthreshold=FATAL &
```

### testing
1.push private message
```sh
$ curl -d "{\"test\":1}" http://localhost:8091/1/admin/push/private?key=Terry-Mao\&expire=600
```
```sh
$ curl -d "{\"test\":1}" http://localhost:8091/admin/push?key=Terry-Mao\&expire=60\&gid=0 (Compatibility with older versions, recommend use above)
```
succeed response：{"ret":0}
* note: the message of new push url must be json format, otherwise it will get a error when invoke ‘get offline message’ url.

2.get offline message

open in browser
```scala
http://localhost:8090/1/msg/get?k=Terry-Mao&m=0
```
```scala
http://localhost:8090/msg/get?key=Terry-Mao&mid=1&pmid=0 (Compatibility with older versions, recommend use above)
```
succeed response:
```json
{
    "data":{
        "msgs":[
            {"msg":{"test":1},"mid":13996474938346192,"gid":0}
        ]
    },
    "ret":0
}
```
succeed response: (Compatibility with older versions)
```json
{
    "data": {
        "msgs": [
            "{\"msg\":{\"test\":1},\"expire\":1391943609703654726,\"mid\":13919435497036558}"
        ],
        "pmsgs": [
            "{\"msg\":{\"test\":1},\"expire\":1391943637016665915,\"mid\":13919435770166656}"
        ]
    },
    "ret": 0
}
```
* node: each of message in new response json is a struct, and the old message is string

3.get node address

open in browser
```scala
http://localhost:8090/1/server/get?k=Terry-Mao&p=2
```
```scala
http://localhost:8090/server/get?key=Terry-Mao&proto=2 (Compatibility with older versions, recommend use above)
```
succeed response:
```json
{
    "data":{
        "server":"localhost:6969"
    },
    "ret":0
}
```

## Configuration
[web](https://github.com/Terry-Mao/gopush-cluster/blob/master/web/web-example.conf)
[comet](https://github.com/Terry-Mao/gopush-cluster/blob/master/comet/comet-example.conf)
[message](https://github.com/Terry-Mao/gopush-cluster/blob/master/message/message-example.conf)

## Examples
java: [gopush-cluster-sdk](https://github.com/Terry-Mao/gopush-cluster-sdk)

ios: [CocoaGoPush](https://github.com/gdier/CocoaGoPush)

javascript: [gopush-cluster-javascript-sdk](https://github.com/Lanfei/gopush-cluster-javascript-sdk)

## Documentation
`web`
[external](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/web/external_proto_en.textile)
[internal](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/web/internal_proto_en.textile)

`comet`
[client](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/comet/client_proto_en.textile)
[internal](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/comet/rpc_proto_en.textile)

`message`
[internal](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/message/rpc_proto_en.textile)

## FAQ
1.install hg
```sh
$ wget http://mercurial.selenic.com/release/mercurial-1.4.1.tar.gz 
$ tar -xvf mercurial-1.4.1.tar.gz
$ cd mercurial-1.4.1
$ make
$ make install
```
* if error couldn`t find ‘Python.h’
```sh
$ yum -y install python-devel
```
* if error：couldn`t find libraries
```sh
$ export PYTHONPATH=/usr/local/lib64/python2.6/site-packages
```
2.install tcl8.5
```sh
$ cd /data/programfiles$ wget http://downloads.sourceforge.net/tcl/tcl8.5.10-src.tar.gz$ tar -xvf tcl8.5.10-src.tar.gz -C ./$ cd tcl8.5.10$ cd unix$ ./configure$ make$ make install
```

## LICENSE
TODO
