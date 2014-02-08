## Terry-Mao/gopush-cluster message 安装文档
#### 通过go get 直接获取或者更新gopush-cluster，及依赖包
```sh
$ go get -u github.com/Terry-Mao/gopush-cluster
$ go get -u github.com/Terry-Mao/goconf
$ go get -u github.com/garyburd/redigo/redis
```

#### build或install message模块
```sh
$ cd ./message
# build：会在当前目录编译生成message可执行文件
$ go build

# install：会在${GOPATH}/bin目录编译生成message可执行文件
$ go install
```

#### 启动message
```sh
# 建议使用绝对路径，根据不同的方式（build，install）找到message可执行文件
$  /usr/bin/nohup ./message -c=./message.conf > /tmp/panic.log 2>&1 &
```

