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
 * private message push
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
$ mkdir -p /data/apps$ mkdir -p /data/logs/gopush-cluster$ mkdir -p /data/programfiles
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
$ wget http://download.redis.io/releases/redis-2.8.17.tar.gz
$ tar -xvf redis-2.8.17.tar.gz -C ./
$ cd redis-2.8.17/src
$ make
$ make test
$ make install
$ mkdir /etc/redis
$ cp /data/programfiles/redis-2.8.17/redis.conf /etc/redis/
$ cp /data/programfiles/redis-2.8.17/src/redis-server /etc/init.d/redis-server
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
1.download(depend on your system, select from [here](http://golang.org/dl/))
```sh
# centos
$ cd /data/programfiles
$ wget -c --no-check-certificate https://go.googlecode.com/files/go1.3.linux-amd64.tar.gz
$ tar -xvf go1.3.linux-amd64.tar.gz -C /usr/local
```
2.golang env
```sh
$ vi /etc/profile.d/golang.sh
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
$ cp log.xml $GOPATH/bin/message_log.xml
$ cd ../comet/
$ go install
$ cp comet-example.conf $GOPATH/bin/comet.conf
$ cp log.xml $GOPATH/bin/comet_log.xml
$ cd ../web/
$ go install
$ cp web-example.conf $GOPATH/bin/web.conf
$ cp log.xml $GOPATH/bin/web_log.xml
```
All done!!!

### start gopush-cluster
```sh
$ cd /$GOPATH/bin
$ nohup $GOPATH/bin/message -c $GOPATH/bin/message.conf 2>&1 >> /data/logs/gopush-cluster/panic-message.log &
$ nohup $GOPATH/bin/comet -c $GOPATH/bin/comet.conf 2>&1 >> /data/logs/gopush-cluster/panic-comet.log &
$ nohup $GOPATH/bin/web -c $GOPATH/bin/web.conf 2>&1 >> /data/logs/gopush-cluster/panic-web.log &
```

If start failed, please check the log from the path configured in log.xml or /data/logs/gopush-cluster/panic-xxx.log

### testing
1.push single private message
```sh
$ curl -d "{\"test\":1}" http://localhost:8091/1/admin/push/private?key=Terry-Mao\&expire=600
```
succeed response: `{"ret":0}`<br>

2.push multiple private messages
```sh
$ curl -d "{\"m\":\"{\\\"test\\\":1}\",\"k\":\"t1,t2,t3\"}" http://localhost:8091/1/admin/push/mprivate?expire=600
```
succeed response: `{"data":{"fk":["t1","t2"]},"ret":0}`<br>
* filed `m` is the push-message, `k` is contain all of push-keys that comma separated<br>

		note:1)the message of new push url must be json format string.
			 2)in the case of push multiple, no `fk` filed, only if respond when part of
			  messages what push failed(`fk` mean failed-keys). it`s a string array structure.

3.get offline message

open in browser
```scala
http://localhost:8090/1/msg/get?k=Terry-Mao&m=0
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
4.get node address

open in browser
```scala
http://localhost:8090/1/server/get?k=Terry-Mao&p=2
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
