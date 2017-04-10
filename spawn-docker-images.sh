echo 'build images based alpine3.5 for gopush-cluster ...'

cd $GOPATH/src/github.com/Terry-Mao/gopush-cluster/message
#go install
#cp message-example.conf $GOPATH/bin/message.conf
#cp log.xml $GOPATH/bin/message_log.xml
MSG_CFG="message-example.conf"
sed -i -e "s/^\s*addr localhost:2181\s*/addr zookeeper:2181/" \
       -e "s/^\s*node1:1 tcp@localhost:6379\s*/node1:1 tcp@redis:6379/" \
       -e "s|^\s*log /data/apps/go/bin/message_log.xml\s*|log /opt/message/message_log.xml|" \
       -e "s|localhost|0\.0\.0\.0|g" \
       $MSG_CFG

go build

cd ../comet/
#go install
#cp comet-example.conf $GOPATH/bin/comet.conf
#cp log.xml $GOPATH/bin/comet_log.xml
COMET_CFG="comet-example.conf"
sed -i -e "s/^\s*addr localhost:2181\s*/addr zookeeper:2181/" \
       -e "s|^\s*log /data/apps/go/bin/comet_log.xml\s*|log /opt/comet/comet_log.xml|" \
       -e "s|localhost|0\.0\.0\.0|g" \
       $COMET_CFG

go build

cd ../web/
#go install
#cp web-example.conf $GOPATH/bin/web.conf
#cp log.xml $GOPATH/bin/web_log.xml
WEB_CFG="web-example.conf"
sed -i -e "s/^\s*addr localhost:2181\s*/addr zookeeper:2181/" \
       -e "s|^\s*log /data/apps/go/bin/web_log.xml\s*|log /opt/web/web_log.xml|" \
       -e "s|localhost|0\.0\.0\.0|g" \
       $WEB_CFG

go build

echo 'done !!!'
