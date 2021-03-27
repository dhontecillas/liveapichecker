
docker:
	docker build -t dhontecillas/liveapichecker:latest -t dhontecillas/liveapichecker:v0.1 .
	docker image prune --filter label=stage=builder
