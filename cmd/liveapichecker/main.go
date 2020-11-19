package main

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

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
}

func launchServer() {
	srv := &http.Server{
		Addr:    conf.Address(),
		Handler: router,
	}

	sigChan := make(chan os.Signal, 1)

	go signalHandler(sigChan, srv, ed)

	if err := srv.ListenAndServe(); err != nil {
		if err != http.ErrServerClosed {
			ed.Insights.L.Err(err, "error launching server")
			ed.Shutdown()
		}
	}
}

type ReqResp struct {
}

func openapiChecker() {
	select {}
}
