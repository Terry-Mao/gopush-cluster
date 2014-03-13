// Copyright © 2014 Terry Mao, LiuDing All rights reserved.
// This file is part of gopush-cluster.

// gopush-cluster is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// gopush-cluster is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with gopush-cluster.  If not, see <http://www.gnu.org/licenses/>.

package main

// 65535 未知错误
// 65534 参数错误
// 0     成功
// 1001  不存在节点

const (
	OK              = 0
	NoNodeErr       = 1001
	NodeExist       = 1002
	UnknownProtocol = 1003
	ParamErr        = 65534
	InternalErr     = 65535

	OKMsg               = "ok"
	NoNodeErrMsg        = "no node"
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
