package rpc

const (
	PrivateGroupID = 0
	PublicGroupID  = 1
)

// Channel Push Private Message Args
type ChannelPushPrivateArgs struct {
	GroupID int    // message group id
	Msg     string // message content
	Expire  int64  // message expire second
	Key     string // subscriber key
}

// Channel Push Public Message Args
type ChannelPushPublicArgs struct {
	MsgID   int64  // message id
	Msg     string // message content
}

// Channel Migrate Args
type ChannelMigrateArgs struct {
	Nodes []string // current comet nodes
	Vnode int      // ketama virtual node number
}

// Channel New Args
type ChannelNewArgs struct {
	Expire int64  // message expire second
	Token  string // auth token
	Key    string // subscriber key
}
