package rpc

// Message Save Args
type MessageSaveArgs struct {
	MsgID  int64  // message id
	Msg    string // message content
	Expire int64  // message expire second
	Key    string // subscriber key
}

// Public Message Save Args
type MessageSavePubArgs struct {
	MsgID  int64  // message id
	Msg    string // message content
	Expire int64  // message expire second
}

// Message Get Args
type MessageGetArgs struct {
	MsgID    int64  // message id
	PubMsgID int64  // public message id
	Key      string // subscriber key
}

// Message Get Response
type MessageGetResp struct {
	Ret     int      // response
	Msgs    []string // messages
	PubMsgs []string // public messages
}
