package main

// 65535 未知错误
// 0     成功
// 1     参数错误
// 1001  不存在节点

const (
	OK          = 0
	ParamErr    = 1
	NoNodeErr   = 1001
	InternalErr = 65535

	OKMsg               = "ok"
	ParamErrMsg         = "param error"
	NoNodeErrMsg        = "node is not alive"
	InvalidSessionIDMsg = "invalid session id"

	InternalErrMsg = "internal exception"
)

var (
	errMsg map[int]string
)

func init() {
	// Err massage
	errMsg = make(map[int]string)
	errMsg[OK] = OKMsg
	errMsg[ParamErr] = ParamErrMsg
	errMsg[NoNodeErr] = NoNodeErrMsg
	errMsg[InternalErr] = InternalErrMsg
}

func GetErrMsg(ret int) string {
	msg, ok := errMsg[ret]
	if ok {
		return msg
	}

	return ""
}
