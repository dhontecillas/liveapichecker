package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/dhontecillas/liveapichecker/pkg/analyzer"
	"github.com/dhontecillas/liveapichecker/pkg/proxy"
	"github.com/spf13/viper"

	"github.com/go-openapi/loads"
	/*
		"github.com/go-openapi/analysis"
		"github.com/go-openapi/errors"
		"github.com/go-openapi/loads"
		"github.com/go-openapi/spec"
		"github.com/go-openapi/strfmt"

		"github.com/go-openapi/runtime"
		"github.com/go-openapi/runtime/logger"
		"github.com/go-openapi/runtime/middleware/untyped"
		"github.com/go-openapi/runtime/security"
	*/)

const (
	OpenAPIFileKey string = "openapi.file"
	ForwardURLKey  string = "forward.url"
)

func main() {
	v := viper.GetViper()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	fileName := v.GetString(OpenAPIFileKey)
	if len(fileName) == 0 {
		panic("cannot read OPENAPI_FILE filename")
	}

	forwardURL := v.GetString(ForwardURLKey)
	if len(forwardURL) == 0 {
		panic("cannot read forward url")
	}
	fmt.Printf("checking %s against %s\n", fileName, forwardURL)

	specDoc, err := loads.Spec(fileName)
	if err != nil {
		fmt.Printf("\nerror in provided spec:\n%s\n", err.Error())
		return
	}

	covChecker := analyzer.NewCoverageChecker(specDoc)

	proxyH := proxy.NewProxyHandler(forwardURL)
	parallelH := proxy.NewParallelHandler(proxyH, covChecker)
	parallelH.LaunchParallelProc()

	launchServer(parallelH)
}

func launchServer(hfn http.Handler) {
	srv := &http.Server{
		// TODO: load this from config:
		Addr:    "0.0.0.0:7777",
		Handler: hfn,
	}

	sigChan := make(chan os.Signal, 1)

	go signalHandler(sigChan, srv)

	if err := srv.ListenAndServe(); err != nil {
		if err != http.ErrServerClosed {
			fmt.Printf("error %s\nSHUTTING DOWN", err.Error())
		}
	}
}

func signalHandler(sc chan os.Signal, srv *http.Server) {
	signal.Notify(sc, os.Interrupt)
	<-sc
	fmt.Printf("\nshutdown signal received\n")
	shutdownServer(srv)
}

func shutdownServer(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

type ReqResp struct {
}

func openapiChecker() {
	select {}
}
