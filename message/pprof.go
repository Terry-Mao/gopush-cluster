package main

import (
	"net/http"
	"net/http/pprof"
)

// StartPprof start http pprof.
func StartPprof() {
	pprofServeMux := http.NewServeMux()
	pprofServeMux.HandleFunc("/debug/pprof/", pprof.Index)
	pprofServeMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	pprofServeMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	pprofServeMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	for _, addr := range Conf.PprofBind {
		go func() {
			Log.Info("start pprof listen addr:\"%s\"", addr)
			if err := http.ListenAndServe(addr, pprofServeMux); err != nil {
				Log.Error("http.ListenAndServe(\"%s\") error(%v)", addr, err)
				panic(err)
			}
		}()
	}
}
