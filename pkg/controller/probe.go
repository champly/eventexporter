package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/klog/v2"
)

func startProbleCheck(ctx context.Context) {
	server := &http.Server{
		Addr: ":80",
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		fmt.Fprint(rw, "ok")
	})
	server.Handler = mux

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if !strings.EqualFold(err.Error(), "http: Server closed") {
				klog.Error(err)
				return
			}
		}
		klog.Info("http shutdown")
	}()
	<-ctx.Done()
	server.Shutdown(context.Background())
}
