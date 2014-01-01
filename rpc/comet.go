package rpc

// Channel Publish Args
type ChannelPublishArgs struct {
	MsgID  int64  // message id
	Msg    string // message content
	Expire int64  // message expire second
	Key    string // subscriber key
}

// Channel Migrate Args
type ChannelMigrateArgs struct {
	Nodes []string // current comet nodes
	Vnode int      // ketama virtual node number
}
