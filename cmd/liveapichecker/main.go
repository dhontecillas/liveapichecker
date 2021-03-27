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
)

const (
	OpenAPIFileKey      string = "liveapichecker.openapi.file"
	ReportFileKey       string = "liveapichecker.report.file"
	ReportServerAddress string = "liveapichecker.report.server.address"
	ForwardToKey        string = "liveapichecker.forward.to"
	ForwardListenAtKey  string = "liveapichecker.forward.listenat"
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
			covChecker.DumpResultsToFile(outFile)
		}
	}
	var reportsSrv *http.Server
	reportsAddress := v.GetString(ReportServerAddress)
	if len(reportsAddress) != 0 {
		reportsSrv = analyzer.LaunchReportsServer(reportsAddress, covChecker)
	}

	proxyH, err := proxy.NewProxyHandler(forwardTo)
	if err != nil {
		fmt.Printf("cannot start proxy to forward to: %s -> %s\n",
			forwardTo, err.Error())
		return
	}
	parallelH := proxy.NewParallelHandler(proxyH, covChecker)
	parallelH.LaunchParallelProc()

	address := v.GetString(ForwardListenAtKey)
	if len(address) == 0 {
		address = "127.0.0.1:7777"
	}
	proxySrv := launchServer(parallelH, address)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	shutdownServer(proxySrv)
	shutdownServer(reportsSrv)

	if dumpCovFn != nil {
		fmt.Printf("post shutdown cleanup\n")
		dumpCovFn()
	}
}

func launchServer(hfn http.Handler, address string) *http.Server {
	srv := &http.Server{
		Addr:    address,
		Handler: hfn,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				fmt.Printf("shutting down...\n")
			} else {
				fmt.Printf("error %s\nSHUTTING DOWN", err.Error())
			}
		}
	}()
	return srv
}

func shutdownServer(srv *http.Server) {
	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
