# version 1.0
# author Terry.Mao
CREATE DATABASE IF NOT EXISTS gopush;
USE gopush;

# private message
# DROP TABLE private_msg;
CREATE TABLE IF NOT EXISTS private_msg (
	id bigint unsigned NOT NULL AUTO_INCREMENT PRIMARY KEY, # auto increment id
	skey varchar(64) NOT NULL, # subscriber key
	mid bigint unsigned NOT NULL, # message id
	ttl bigint NOT NULL, # message expire second
	msg blob NOT NULL, # message content
	ctime timestamp NOT NULL DEFAULT '0000-00-00 00:00:00', # create time
	mtime timestamp NOT NULL DEFAULT '0000-00-00 00:00:00', # modify time,
	UNIQUE KEY ux_private_msg_1 (skey, mid),
	INDEX ix_private_msg_1 (ttl)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;