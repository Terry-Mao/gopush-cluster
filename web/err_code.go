package main

// 65535 未知错误
// 65534 参数错误
// 0     成功
// 1001  不存在节点

const (
	OK          = 0
	NoNodeErr   = 1001
	NodeExist   = 1002
	ParamErr    = 65534
	InternalErr = 65535

	OKMsg               = "ok"
	NoNodeErrMsg        = "node is not alive"
	ParamErrMsg         = "param error"
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
	errMsg[NoNodeErr] = NoNodeErrMsg
	errMsg[ParamErr] = ParamErrMsg
	errMsg[InternalErr] = InternalErrMsg
}

func GetErrMsg(ret int) string {
	msg, ok := errMsg[ret]
	if ok {
		return msg
	}

	return ""
}
