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

package rpc

const (
    // common
	// ok
	OK = 0
	// param error
	ParamErr = 65534

    // comet
	// create channel failed
	CreateChannelErr = 2000
	// get channel failed
	GetChannelErr = 2001
	// message push failed
	PushMsgErr = 2002
	// migrate failed
	MigrateErr = 2003
	// add token
	AddTokenErr = 2004
)
