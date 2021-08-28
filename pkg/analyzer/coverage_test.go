package analyzer_test

import (
	"net/http"
	"testing"

	"github.com/dhontecillas/liveapichecker/pkg/analyzer"
	"github.com/dhontecillas/liveapichecker/pkg/proxy"
	"github.com/go-openapi/loads"
	"github.com/stretchr/testify/require"
)

func TestCoverageChecker(t *testing.T) {
	r := require.New(t)
	doc, err := loads.Analyzed([]byte(testSwagger), "2.0")
	r.Nil(err, "swagger file")

	req, _ := http.NewRequest("GET", "https://petstore.swagger.io/v1/pets/?limit=5", nil)
	rwr := proxy.NewResponseWriterRecorder(req)
	rwr.StatusCode = 200

	cc := analyzer.NewCoverageChecker(doc)
	cc.ProcessRecordedResponse(rwr)
	report := cc.Report()

	var getPetsEP *analyzer.EndpointCoverage = nil
	for _, rep := range report {
		if rep.Path == "/pets" {
			getPetsEP = rep
			break
		}
	}

	r.NotNil(getPetsEP)

	nSC := len(getPetsEP.StatusCodes)
	r.Equal(nSC, 1, "want 1 status code, got %d", nSC)

	r.NotNil(getPetsEP.Params)
	params := getPetsEP.Params.Params

	r.Equal(len(params), 1, "should have covered the limit param")
	r.Equal(params[0].Name, "limit")
	r.Equal(len(params[0].CoveredValues), 1, "params %#v", params)
	r.Equal(params[0].CoveredValues[0], "5")
}
