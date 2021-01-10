# Live API Checker

API Proxy to keep track of the endpoints that have
been called along with its responses.

## How it works

The live API checker gets an OpenAPI 2.0 spec as input, and
spins up a proxy server that records the request / responses
to check which endpoints are called along with responses to
keep track of the coverage, and generates a json report
on proxy shutdown.

The main purpose is to keep track of the coverage of API
end to end tests.


## Environment vars configuration


```bash

LIVEAPICHECKER_OPENAPI_FILE="/path/to/openapi/file"
LIVEAPICHECKER_REPORT_FILE="/path/to/report/file.json"
LIVEAPICHECKER_FORWARD_TO="http://localhost:8000"
LIVEAPICHECKER_FORWARD_LISTENAT="locahost:7777"

```

## TODO

### Check compliance with OpenAPI spec and emit report on mismatch

A part from checking the API codes, since we have the recorded
response, we can check that matches the response spec.

### Coverage of parameters

Take into account query params and check the coverage for a set of
different values. Is hard to determine the possible values for
each param, so parameters can be provided to give a default number
of diferent values that each type of param (string, numbers, ...)
should check to give it a "coverage score".

#### Combinations

Problem with previous approach is that different combinations of
params could have different effects on the endpoint, but the number
of combinations can explode even with a low number of different inputs.
