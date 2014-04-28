# version 1.0
# author Terry.Mao
CREATE DATABASE IF NOT EXISTS gopush;
USE gopush;

# offline message
# DROP TABLE message;
CREATE TABLE IF NOT EXISTS message (
	id bigint unsigned NOT NULL AUTO_INCREMENT PRIMARY KEY, # auto increment id
	sub varchar(64) NOT NULL, # subscriber key
	mid bigint unsigned NOT NULL, # message id
	gid int unsigned NOT NULL, # message group id
	expire bigint NOT NULL, # message expire second
	msg blob NOT NULL, # message content
	ctime timestamp NOT NULL DEFAULT '0000-00-00 00:00:00', # create time
	mtime timestamp NOT NULL DEFAULT '0000-00-00 00:00:00', # modify time,
	UNIQUE KEY ux_message_1 (sub, mid),
	INDEX ix_message_1 (expire)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;