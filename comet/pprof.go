package main

import (
	"net/http"
	"net/http/pprof"
)

// StartPprofHttp start http pprof
func StartPprofHttp() error {
	pprofServeMux := http.NewServeMux()
	pprofServeMux.HandleFunc("/debug/pprof/", pprof.Index)
	pprofServeMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	pprofServeMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	pprofServeMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	Log.Info("start listen pprof addr:%s", Conf.PprofAddr)
	err := http.ListenAndServe(Conf.PprofAddr, pprofServeMux)
	if err != nil {
		Log.Error("http.ListenAdServe(\"%s\") failed (%s)", Conf.PprofAddr, err.Error())
		return err
	}

	return nil
}
