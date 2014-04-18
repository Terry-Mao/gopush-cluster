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

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/Terry-Mao/gopush-cluster/hash"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

const (
	defaultMYSQLNode = "node1"
	saveSQL          = "INSERT INTO message(sub,gid,mid,expire,msg,ctime,mtime) VALUES(?,?,?,?,?,?,?)"
	getSQL           = "SELECT msg FROM message WHERE sub=? AND mid>? AND expire>?"
	delExpireSQL     = "DELETE FROM message WHERE expire<=?"
)

var (
	MYSQLNoDBErr = errors.New("can't get a mysql db")
)

// MySQL Storage struct
type MYSQLStorage struct {
	// n
	Pool   map[string]*sql.DB
	Ketama *hash.Ketama
}

// NewMYSQL initialize mysql pool and consistency hash ring
func NewMYSQL() *MYSQLStorage {
	dbPool := make(map[string]*sql.DB)
	for n, source := range Conf.DBSource {
		db, err := sql.Open("mysql", source)
		if err != nil {
			Log.Error("sql.Open(\"mysql\", %s) failed (%v)", source, err)
			panic(err)
		}

		dbPool[n] = db
	}

	s := &MYSQLStorage{Pool: dbPool, Ketama: hash.NewKetama(len(dbPool), 255)}
	go s.delLoop()
	return s
}

// Save implements the Storage Save method.
func (s *MYSQLStorage) Save(key string, msg *Message, mid int64) error {
	db := s.getConn(key)
	if db == nil {
		return MYSQLNoDBErr
	}

	message, _ := json.Marshal(*msg)
	now := time.Now().Unix()
	_, err := db.Exec(saveSQL, key, 0, mid, msg.Expire, string(message), now, now)
	if err != nil {
		Log.Error("db.Exec(%s,%s,%d,%d,%d,%s,now,now) failed (%v)", saveSQL, key, 0, mid, msg.Expire, string(message), now, now, err)
		return err
	}

	return nil
}

// Get implements the Storage Get method.
func (s *MYSQLStorage) Get(key string, mid int64) ([]string, error) {
	db := s.getConn(key)
	if db == nil {
		return nil, MYSQLNoDBErr
	}

	var msg []string
	now := time.Now().Unix()
	rows, err := db.Query(getSQL, key, mid, now)
	if err != nil {
		Log.Error("db.Query(%s,%s,%d,now) failed (%v)", getSQL, key, mid, err)
		return nil, err
	}

	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			Log.Error("rows.Scan() failed (%v)", err)
			return nil, err
		}
		msg = append(msg, m)
	}

	return msg, nil
}

// DelMulti implements the Storage DelMulti method.
func (s *MYSQLStorage) DelMulti(info *DelMessageInfo) error {
	// WARN: nothing to do, cause delete operation run loop periodically
	return nil
}

// DelKey implements the Storage DelKey method.
func (s *MYSQLStorage) DelKey(key string) error {
	// WARN: nothing to do, cause delete operation run loop periodically
	return nil
}

// DelAllExpired enumerate the nodes then delete all expired message.
func (s *MYSQLStorage) DelAllExpired() (int64, error) {
	var affect int64
	now := time.Now().Unix()
	for _, db := range s.Pool {
		res, err := db.Exec(delExpireSQL, now)
		if err != nil {
			Log.Error("db.Exec(%s,now) failed (%v)", delExpireSQL, err)
			return 0, err
		}
		aff, err := res.RowsAffected()
		if err != nil {
			Log.Error("db.Exec(%s,now) failed (%v)", delExpireSQL, err)
			return 0, err
		}
		affect += aff
	}

	return affect, nil
}

// getConn get the connection of matching with key using ketama hash
func (s *MYSQLStorage) getConn(key string) *sql.DB {
	node := defaultMYSQLNode
	if len(s.Pool) > 1 {
		node = s.Ketama.Node(key)
	}

	p, ok := s.Pool[node]
	if !ok {
		Log.Warn("no exists key:\"%s\" in mysql pool", key)
		return nil
	}

	Log.Debug("key:\"%s\", hit node:\"%s\"", key, node)
	return p
}

// delLoop delete expired messages peroridly.
func (s *MYSQLStorage) delLoop() {
	for {
		affect, err := s.DelAllExpired()
		if err != nil {
			Log.Error("delete all of expired messages failed (%v)", err)
			time.Sleep(Conf.MYSQLDelLoopTime)
			continue
		}

		Log.Info("delete all of expired messages OK, count:\"%d\"", affect)
		time.Sleep(Conf.MYSQLDelLoopTime)
	}
}
