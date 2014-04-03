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

const (
	StorageTypeRedis = "redis"
	StorageTypeMysql = "mysql"
)

// Struct for delele message
type DelMessageInfo struct {
	Key  string
	Msgs []string
}

var UseStorage Storage

// Stored messages interface
type Storage interface {
	// Save message
	Save(key string, msg *Message, mid int64) error
	// Get messages
	Get(key string, mid int64) ([]string, error)
	// Delete key
	DelKey(key string) error
	// Delete multiple messages
	DelMulti(info *DelMessageInfo) error
}

func InitStorage() error {
	if Conf.StorageType == StorageTypeRedis {
		UseStorage = NewRedis()
	} else if Conf.StorageType == StorageTypeMysql {
		UseStorage = NewMYSQL()
	}

	return nil
}
