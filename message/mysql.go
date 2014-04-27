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
	"github.com/golang/glog"
	"time"
)

const (
	defaultMYSQLNode = "node1"
	saveSQL          = "INSERT INTO message(sub,gid,mid,expire,msg,ctime,mtime) VALUES(?,?,?,?,?,?,?)"
	getSQL           = "SELECT mid, gid, msg FROM message WHERE sub=? AND mid>? AND expire>?"
	delExpireSQL     = "DELETE FROM message WHERE expire<=?"
)

var (
	ErrNoMySQLConn = errors.New("can't get a mysql db")
)

// MySQL Storage struct
type MySQLStorage struct {
	// n
	Pool   map[string]*sql.DB
	Ketama *hash.Ketama
}

// NewMYSQL initialize mysql pool and consistency hash ring
func NewMYSQL() *MySQLStorage {
	dbPool := make(map[string]*sql.DB)
	for n, source := range Conf.DBSource {
		db, err := sql.Open("mysql", source)
		if err != nil {
			glog.Errorf("sql.Open(\"mysql\", %s) failed (%v)", source, err)
			panic(err)
		}
		dbPool[n] = db
	}
	s := &MySQLStorage{Pool: dbPool, Ketama: hash.NewKetama(len(dbPool), 255)}
	go s.delLoop()
	return s
}

// Save implements the Storage Save method.
func (s *MySQLStorage) Save(key string, msg json.RawMessage, mid int64, gid int, expire uint) error {
	db := s.getConn(key)
	if db == nil {
		return ErrNoMySQLConn
	}
	now := time.Now()
	_, err := db.Exec(saveSQL, key, gid, mid, expire, msg, now, now)
	if err != nil {
		glog.Errorf("db.Exec(%s,%s,%d,%d,%d,%s,now,now) failed (%v)", saveSQL, key, 0, mid, expire, string(msg), err)
		return err
	}
	return nil
}

// Get implements the Storage Get method.
func (s *MySQLStorage) Get(key string, mid int64) ([]*Message, error) {
	db := s.getConn(key)
	if db == nil {
		return nil, ErrNoMySQLConn
	}
	now := time.Now().Unix()
	rows, err := db.Query(getSQL, key, mid, now)
	if err != nil {
		glog.Errorf("db.Query(%s,%s,%d,now) failed (%v)", getSQL, key, mid, err)
		return nil, err
	}
	msgs := []*Message{}
	for rows.Next() {
		m := &Message{}
		if err := rows.Scan(&m.MsgId, &m.GroupId, &m.Msg); err != nil {
			glog.Errorf("rows.Scan() failed (%v)", err)
			return nil, err
		}
		msgs = append(msgs, m)
	}

	return msgs, nil
}

// DelMulti implements the Storage DelMulti method.
func (s *MySQLStorage) DelMulti(info *DelMessageInfo) error {
	// WARN: nothing to do, cause delete operation run loop periodically
	return nil
}

// DelKey implements the Storage DelKey method.
func (s *MySQLStorage) DelKey(key string) error {
	// WARN: nothing to do, cause delete operation run loop periodically
	return nil
}

// DelAllExpired enumerate the nodes then delete all expired message.
func (s *MySQLStorage) DelAllExpired() (int64, error) {
	var affect int64
	now := time.Now().Unix()
	for _, db := range s.Pool {
		res, err := db.Exec(delExpireSQL, now)
		if err != nil {
			glog.Errorf("db.Exec(%s,now) failed (%v)", delExpireSQL, err)
			return 0, err
		}
		aff, err := res.RowsAffected()
		if err != nil {
			glog.Errorf("db.Exec(%s,now) failed (%v)", delExpireSQL, err)
			return 0, err
		}
		affect += aff
	}

	return affect, nil
}

// getConn get the connection of matching with key using ketama hash
func (s *MySQLStorage) getConn(key string) *sql.DB {
	node := defaultMYSQLNode
	if len(s.Pool) > 1 {
		node = s.Ketama.Node(key)
	}
	p, ok := s.Pool[node]
	if !ok {
		glog.Warningf("no exists key:\"%s\" in mysql pool", key)
		return nil
	}
	glog.V(1).Infof("key:\"%s\", hit node:\"%s\"", key, node)
	return p
}

// delLoop delete expired messages peroridly.
func (s *MySQLStorage) delLoop() {
	for {
		affect, err := s.DelAllExpired()
		if err != nil {
			glog.Errorf("delete all of expired messages failed (%v)", err)
			time.Sleep(Conf.MYSQLDelLoopTime)
			continue
		}

		glog.Infof("delete all of expired messages OK, count:\"%d\"", affect)
		time.Sleep(Conf.MYSQLDelLoopTime)
	}
}
