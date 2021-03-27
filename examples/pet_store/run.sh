LOCAL_IP=$(ip address | grep 192 | head -1 |  sed -re 's/.*inet (192\.[0-9]+\.[0-9]+\.[0-9]+).*/\1/g')

docker run -v $(pwd):/data \
    --name petstore_liveapicheck \
    -p 7777:7777 \
    -p 7778:7778 \
    -e LIVEAPICHECKER_OPENAPI_FILE="/data/petstore.yaml" \
    -e LIVEAPICHECKER_REPORT_FILE="/data/report.out.json" \
    -e LIVEAPICHECKER_FORWARD_TO="http://$LOCAL_IP:9876" \
    -e LIVEAPICHECKER_FORWARD_LISTENAT="0.0.0.0:7777" \
    -e LIVEAPICHECKER_REPORT_SERVER_ADDRESS="0.0.0.0:7778" \
    liveapichecker:latest
