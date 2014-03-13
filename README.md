gopush-cluster
==============
gopush-cluster is a go push server cluster.

## Features
 * light weight
 * high performance
 * pure golang implementation
 * message expired
 * offline message store
 * public message or private message push
 * multiple subscribers (can restrict max subscribers)
 * heartbeat（service heartbeat or tcp keepalive）
 * auth (if a subscriber not auth then can't connect to comet node)
 * multiple protocol (websocket, tcp, todo http longpolling)
 * stat
 * cluster support (easy add or remove comet & web & message node)
 * failover support (zookeeper)

## Architecture
 ![gopush-cluster](http://raw.github.com/Terry-Mao/gopush-cluster/master/wiki/architecture/architecture.jpg "gopush-cluster architecture")

## Document
[English](https://github.com/Terry-Mao/gopush-cluster/blob/master/README_en.md)

[中文](https://github.com/Terry-Mao/gopush-cluster/blob/master/README_zh.md)

## LICENSE
gopush-cluster is is distributed under the terms of the GNU General Public License, version 3.0 [GPLv3](http://www.gnu.org/licenses/gpl.txt)
