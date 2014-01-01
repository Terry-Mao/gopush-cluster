package rpc

// Channel Publish Args
type ChannlePubArgs struct {
	MsgID  int64  // message id
	Msg    string // message content
	Expire int64  // message expire second
	Key    string // subscriber key
}
