
docker:
	docker build -t dhontecillas/liveapichecker:latest -t dhontecillas/liveapichecker:v0.1 .
	docker image prune --filter label=stage=builder

dockeralpine:
	docker build -f Dockerfile.alpine -t dhontecillas/liveapichecker-alpine:latest -t dhontecillas/liveapichecker-alpine:v0.1 .
	docker image prune --filter label=stage=builder
