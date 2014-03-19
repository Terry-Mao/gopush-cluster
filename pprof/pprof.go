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

package pprof

import (
	. "github.com/Terry-Mao/gopush-cluster/log"
	"net/http"
	netPprof "net/http/pprof"
)

// StartPprof start http pprof.
func StartPprof(pprofBind []string) {
	pprofServeMux := http.NewServeMux()
	pprofServeMux.HandleFunc("/debug/pprof/", netPprof.Index)
	pprofServeMux.HandleFunc("/debug/pprof/cmdline", netPprof.Cmdline)
	pprofServeMux.HandleFunc("/debug/pprof/profile", netPprof.Profile)
	pprofServeMux.HandleFunc("/debug/pprof/symbol", netPprof.Symbol)
	for _, addr := range pprofBind {
		go func() {
			Log.Info("start pprof listen addr:\"%s\"", addr)
			if err := http.ListenAndServe(addr, pprofServeMux); err != nil {
				Log.Error("http.ListenAdServe(\"%s\") error(%v)", addr, err)
				panic(err)
			}
		}()
	}
}
