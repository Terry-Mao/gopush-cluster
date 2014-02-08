## Terry-Mao/gopush-cluster comet 安装文档
#### 通过go get 直接获取或者更新gopush-cluster
```sh
$ go get -u github.com/Terry-Mao/gopush-cluster/comet
```

#### build或install comet模块
```sh
$ cd ./comet
# build：会在当前目录编译生成comet可执行文件
$ go build

# install：会在${GOPATH}/bin目录编译生成comet可执行文件
$ go install
```

#### 启动comet
```sh
# 建议使用绝对路径，根据不同的方式（build，install）找到comet可执行文件
$  /usr/bin/nohup ./comet -c=./comet-examplex.conf > /tmp/panic.log 2>&1 &
```
