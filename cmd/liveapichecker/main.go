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
	OpenAPIFileKey     string = "liveapichecker.openapi.file"
	ReportFileKey      string = "liveapichecker.report.file"
	ForwardToKey       string = "liveapichecker.forward.to"
	ForwardListenAtKey string = "liveapichecker.forward.listenat"
)

func main() {
	v := viper.GetViper()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	fileName := v.GetString(OpenAPIFileKey)
	if len(fileName) == 0 {
		panic("cannot read OPENAPI_FILE filename")
	}

	forwardTo := v.GetString(ForwardToKey)
	if len(forwardTo) == 0 {
		panic("cannot read forward to")
	}
	fmt.Printf("checking %s against %s\n", fileName, forwardTo)

	specDoc, err := loads.Spec(fileName)
	if err != nil {
		fmt.Printf("\nerror in provided spec:\n%s\n", err.Error())
		return
	}

	covChecker := analyzer.NewCoverageChecker(specDoc)
	var dumpCovFn func()
	outFile := v.GetString(ReportFileKey)
	if len(outFile) > 0 {
		dumpCovFn = func() {
			fmt.Printf("\ndumpCov called \n\n")
			covChecker.DumpResultsToFile(outFile)
		}
		fmt.Printf("\ndumpCovFn is not null %#v\n", dumpCovFn)
	} else {
		fmt.Printf("\ndumpCovFn IS NULL %#v\n", dumpCovFn)
	}

	proxyH := proxy.NewProxyHandler(forwardTo)
	parallelH := proxy.NewParallelHandler(proxyH, covChecker)
	parallelH.LaunchParallelProc()

	address := v.GetString(ForwardListenAtKey)
	if len(address) == 0 {
		address = "127.0.0.1:7777"
	}
	launchServer(parallelH, address, dumpCovFn)
}

func launchServer(hfn http.Handler, address string, postShutdownFn func()) {
	srv := &http.Server{
		Addr:    address,
		Handler: hfn,
	}

	sigChan := make(chan os.Signal, 1)

	go signalHandler(sigChan, srv)

	if err := srv.ListenAndServe(); err != nil {
		if err != http.ErrServerClosed {
			fmt.Printf("error %s\nSHUTTING DOWN", err.Error())
		}
	}

	if postShutdownFn != nil {
		postShutdownFn()
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
