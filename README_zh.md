gopush-cluster
==============
`Terry-Mao/gopush-cluster` 是一个支持集群的comet服务（支持websocket，和tcp协议）。

---------------------------------------
  * [特性](#特性)
  * [安装](#安装)
  * [使用](#使用)
  * [配置](#配置)
  * [例子](#例子)
  * [文档](#文档)
  * [更多](#更多)

---------------------------------------

## 特性
 * 轻量级
 * 高性能
 * 纯Golang实现
 * 支持消息过期
 * 支持离线消息存储
 * 支持单个以及多个私信推送
 * 支持单个Key多个订阅者（可限制订阅者最大人数）
 * 心跳支持（应用心跳和tcp keepalive）
 * 支持安全验证（未授权用户不能订阅）
 * 多协议支持（websocket，tcp）
 * 详细的统计信息
 * 可拓扑的架构（支持增加和删除comet节点，web节点，message节点）
 * 利用Zookeeper支持故障转移

## 安装(版本1.0.5)
### 一、安装依赖
```sh
$ yum -y install java-1.7.0-openjdk$ yum -y install gcc-c++
```

### 二、搭建zookeeper
1.新建目录
```sh
$ mkdir -p /data/apps$ mkdir -p /data/logs/gopush-cluster$ mkdir -p /data/programfiles
```

2.下载[zookeeper](http://www.apache.org/dyn/closer.cgi/zookeeper/)，推荐下载3.4.5或更高版本
```sh
$ cd /data/programfiles
$ wget http://mirror.bit.edu.cn/apache/zookeeper/zookeeper-3.4.5/zookeeper-3.4.5.tar.gz
$ tar -xvf zookeeper-3.4.5.tar.gz -C ./
```
3.启动zookeeper(zookeeper的集群配置在这里不做详细介绍,如果有多台机器,建议做集群)
```sh
$ cp /data/programfiles/zookeeper-3.4.5/conf/zoo_sample.cfg /data/programfiles/zookeeper-3.4.5/conf/zoo.cfg
$ cd /data/programfiles/zookeeper-3.4.5/bin
$ ./zkServer.sh start
```
### 三、搭建redis
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
* 如果如下报错,则安装tcl8.5(参考附资料2)
```sh
which: no tclsh8.5 in (/usr/lib64/qt-3.3/bin:/usr/local/bin:/usr/bin:/bin:/usr/local/sbin:/usr/sbin:/sbin:/home/geffzhang/bin)
You need 'tclsh8.5' in order to run the Redis test
Make[1]: *** [test] error 1
make[1]: Leaving directory ‘/data/program files/redis-2.6.4/src’
Make: *** [test] error 2！
```
### 四、安装git工具（如果已安装则可跳过此步）
参考：[git](http://git-scm.com/download/linux)
```sh
$ yum -y install git
```
### 五、搭建golang环境
1.下载源码(根据自己的系统下载对应的[安装包](http://golang.org/dl/))
```sh
$ cd /data/programfiles
$ wget -c --no-check-certificate https://go.googlecode.com/files/go1.3.linux-amd64.tar.gz
$ tar -xvf go1.3.linux-amd64.tar.gz -C /usr/local
```
2.配置GO环境变量
(这里我加在/etc/profile.d/golang.sh)
```sh
$ vi /etc/profile.d/golang.sh
# 将以下环境变量添加到profile最后面
export GOROOT=/usr/local/go
export PATH=$PATH:$GOROOT/bin
export GOPATH=/data/apps/go
$ source /etc/profile
```
### 六、部署gopush-cluster
1.下载gopush-cluster及依赖包
```sh
$ ./dependencies.sh
```
* 如果提示如下,说明需要安装谷歌的hg工具（安装mercurial,参考附资料1）

		go: missing Mercurial command. See http://golang.org/s/gogetcmd
		package code.google.com/p/go.net/websocket: exec: "hg": executable file not found in $PATH

2.安装message、comet、web模块(配置文件请依据实际机器环境配置)
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
到此所有的环境都搭建完成！
### 七、启动gopush-cluster
```sh
$ cd /$GOPATH/bin
$ nohup $GOPATH/bin/message -c $GOPATH/bin/message.conf 2>&1 >> /data/logs/gopush-cluster/panic-message.log &
$ nohup $GOPATH/bin/comet -c $GOPATH/bin/comet.conf 2>&1 >> /data/logs/gopush-cluster/panic-comet.log &
$ nohup $GOPATH/bin/web -c $GOPATH/bin/web.conf 2>&1 >> /data/logs/gopush-cluster/panic-web.log &
```

如果启动失败，可通过log.xml里找到日志路径或者/data/logs/gopush-cluster/下的panic-xxx.log，通过日志来排查问题.

### 八、测试
1.推送单个私信（例：消息过期时间为expire=600秒）
```sh
$ curl -d "{\"test\":1}" http://localhost:8091/1/admin/push/private?key=Terry-Mao\&expire=600
```
成功返回：`{"ret":0}`

2.批量推送私信
```sh
$ curl -d "{\"m\":\"{\\\"test\\\":1}\",\"k\":\"t1,t2,t3\"}" http://localhost:8091/1/admin/push/mprivate?expire=600
```
成功返回：`{"data":{"fk":["t1","t2"]},"ret":0}`<br>
* 字段`m`是消息体，`k`是要批量推送的订阅key，每个key用`,`分割。<br>

		注:1)新版推送的消息内容必须是json格式，否则获取消息时会报错.
		   2)批量推送正常情况下是没有`fk`字段的,如果有部分推送失败则返回`fk`，结构为字符串数组.

3.获取离线消息接口

在浏览器中打开：
```scala
http://localhost:8090/1/msg/get?k=Terry-Mao&m=0
```
成功返回:
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

4.获取节点接口

在浏览器中打开：
```scala
http://localhost:8090/1/server/get?k=Terry-Mao&p=2
```
成功返回：
```json
{
    "data":{
        "server":"localhost:6969"
    },
    "ret":0
}
```
### 九、附资料
1.下载安装[hg](code.google.com/p/go.net/websocket)
```sh
$ wget http://mercurial.selenic.com/release/mercurial-1.4.1.tar.gz
$ tar -xvf mercurial-1.4.1.tar.gz
$ cd mercurial-1.4.1
$ make
$ make install
```
* 如果安装提示找不到文件‘Python.h’ 则需要安装 python-devel
```sh
$ yum -y install python-devel
```
* 如果报错：couldn`t find libraries,则添加环境变量
```sh
$ export PYTHONPATH=/usr/local/lib64/python2.6/site-packages
```
2.安装tcl8.5
```sh
$ cd /data/programfiles
$ wget http://downloads.sourceforge.net/tcl/tcl8.5.10-src.tar.gz
$ tar -xvf tcl8.5.10-src.tar.gz -C ./
$ cd tcl8.5.10
$ cd unix
$ ./configure
$ make
$ make install
```

## docker 部署

### 准备
1. 配置本地 go 环境, 假设  GOPATH=~/go-workspace
2. 安装 Docker (安装最新版的docker， docker-compose), mac 下可以直接安装最新稳定版的 docker-for-mac.
3. 下载源码并安装依赖(./dependencies.sh)
4. 修改源码(这里用docker-compose 一键部署)

```sh
$ cd ~/go-workspace/src/github.com/Terry-Mao/gopush-cluster
step1. 修改 comet/zk.go, 从52行开始注释三行，添加三行

52	// nodeInfo.RpcAddr = Conf.RPCBind
53	// nodeInfo.TcpAddr = Conf.TCPBind
54	// nodeInfo.WsAddr = Conf.WebsocketBind
++  nodeInfo.RpcAddr = []string{"comet:6970", "comet:7070"}
++  nodeInfo.TcpAddr = []string{"comet:6969", "comet:7069"}
++	nodeInfo.WsAddr = []string{"comet:6968", "comet:7068"}


step2. 修改 message/zk.go, 从43行开始注释一行行，添加行

43// nodeInfo.Rpc = Conf.RPCBind
++	nodeInfo.Rpc = []string{"message:8070", "message:8270"}

```

5. 开始部署

```sh
$ docker pull robertzhouxh/gobuild-alpine-3.5
$ docker pull robertzhouxh/redis3.2-alpine
$ docker pull wurstmeister/zookeeper

#################### 在 robertzhouxh/gobuild-alpine-3.5 (go 版本为1.7.5) 容器内编译 comet， web， message 可执行文件 #######################
$ docker run -it --rm -v ~/go-workspace:/go robertzhouxh/gobuild-alpine-3.5 bash
$ cd /go/src/github.com/Terry-Mao/gopush-cluster
$ ./spawn-docker-images.sh
$ exit

#################### 在宿主机内 #######################
cd ~/go-workspace/src/github.com/Terry-Mao/gopush-cluster
docker-compose build
docker-compose up 或者 docker-compose up -d
```

部署完毕！

test:
```sh
$ curl -d "{\"test\":1}" http://localhost:8091/1/admin/push/private?key=Terry-Mao\&expire=600
{"ret":0}
```

### 说明
1. 镜像全部基于 alpine3.5 制作， 保证镜像最小化。
2. 要修改两个源码文件的绑定地址， 来适配 docker-compose 集群内部通信。
3. 尽量使用docker， docker-compose 最新版。

## 配置
### web节点的配置文件示例：
[web](https://github.com/Terry-Mao/gopush-cluster/blob/master/web/web-example.conf)

### comet节点的配置文件示例：
[comet](https://github.com/Terry-Mao/gopush-cluster/blob/master/comet/comet-example.conf)

### message节点的配置文件示例：
[message](https://github.com/Terry-Mao/gopush-cluster/blob/master/message/message-example.conf)

## 例子
java: [gopush-cluster-sdk](https://github.com/Terry-Mao/gopush-cluster-sdk)

ios: [CocoaGoPush](https://github.com/gdier/CocoaGoPush)

javascript: [gopush-cluster-javascript-sdk](https://github.com/Lanfei/gopush-cluster-javascript-sdk)

## 文档
### web节点相关的文档：
[内部协议](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/web/internal_proto_zh.textile)主要针对内部管理如推送消息、管理comet节点等。

[客户端协议](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/web/external_proto_zh.textile)主要针对客户端使用，如获取节点、获取离线消息等。
### comet节点相关的文档：
[客户端协议](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/comet/client_proto_zh.textile)主要针对客户端连接comet节点的协议说明。

[内部RPC协议](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/comet/rpc_proto_zh.textile)主要针对内部RPC接口使用的说明。
### message节点的相关文档：
[内部RPC协议](https://github.com/Terry-Mao/gopush-cluster/blob/master/wiki/message/rpc_proto_zh.textile)主要针对内部RPC接口的使用说明。

## 更多
TODO
