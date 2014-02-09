## Terry-Mao/gopush-cluster web 安装文档
#### 通过go get 直接获取或者更新gopush-cluster，及依赖包
```sh
$ go get -u github.com/Terry-Mao/gopush-cluster
$ go get -u github.com/Terry-Mao/goconf
$ go get -u launchpad.net/gozk/zookeeper
```

#### build或install web模块
```sh
$ cd ./web
# build：会在当前目录编译生成web可执行文件
$ go build

# install：会在${GOPATH}/bin目录编译生成web可执行文件
$ go install

# web模块依赖zookeeper，该服务端配置不在此详写.
# 请参考：http://zookeeper.apache.org/releases.html,推荐下载3.4.5版本
```

#### 启动web
```sh
# 建议使用绝对路径，根据不同的方式（build，install）找到web可执行文件
$  /usr/bin/nohup ./web -c=./web.conf > /tmp/panic.log 2>&1 &
```


