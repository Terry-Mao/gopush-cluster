## Terry-Mao/gopush-cluster Comet 协议文档
直接使用Redis的协议格式，方便解析和使用，参考[redis protocol](http://redis.io/topics/protocol)

## 请求
格式参照上面提到的Redis协议来传递参数，参数列表：
(head). | 字段 | 描述 | 顺序 | 
| "cmd" | 指令，发起订阅指令为“sub” | 0 |
| "key" | 用户发起订阅的KEY值 | 1 |
| "mid" | 用户本地保存的“最后收到的消息ID”，如果没有就用0 | 2 |
| "heartbeat" | 长连接的心跳周期（单位：秒）| 3 |

例如：
`*4\r\n$3\r\nsub\r\n$9\r\nTerry-Mao\r\n$1\r\n0\r\n$2\r\n10`

## 响应
格式参照上面提到的Redis协议来返回reply
例如：
`$5\r\nTerry`
其中Terry就是推送的消息内容

