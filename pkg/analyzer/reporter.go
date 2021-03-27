package analyzer

import (
	"fmt"
	"net/http"
)

func LaunchReportsServer(address string,
	covChk *CoverageChecker) *http.Server {

	rh := NewReportsHandler(covChk)

	srv := &http.Server{
		Addr:    address,
		Handler: rh,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				fmt.Printf("shutting down reports server...\n")
			} else {
				fmt.Printf("error %s\nSHUTTING DOWN REPORTS SERVER\n", err.Error())
			}
		}
	}()

	return srv
}

type ReportsHandler struct {
	covChk *CoverageChecker
}

func NewReportsHandler(covChk *CoverageChecker) *ReportsHandler {
	return &ReportsHandler{
		covChk: covChk,
	}
}

func (rh *ReportsHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h := rw.Header()
	h.Add("Content-Type", "application/json")
	res, err := rh.covChk.DumpResultsToJSONString()
	if err != nil {
		rw.WriteHeader(http.StatusBadGateway)
		rw.Write([]byte(err.Error()))
	}
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(res))
}
