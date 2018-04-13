BUILD=edge-for-image
CODE=edge_server.go
HOST=root@49.4.5.210

.PHONY: build
build:
    GOOS=linux go build -o $(BUILD) $(CODE)

rerun: build
    @-pkill edge-for-image
    ./start-edge-local.sh

deploy: build
    @-ssh $(HOST) pkill edge-for-image
    scp edge-for-image $(HOST):/opt/edge-for-image/
    ./start-edge-remote.sh


.PHONY: clean
clean:
    rm -rf $(BUILD)