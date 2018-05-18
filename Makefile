BUILD=edge-for-image
CODE=edge_server.go
HOST=root@49.4.5.210
REMOTE_DIR=/opt/Golang/src/edge-for-image

.PHONY: build
build:
	GOOS=linux go build -o $(BUILD) $(CODE)

rerun: build
	@-pkill edge-for-image
	./start-edge-local.sh

deploy: build
	@-ssh $(HOST) pkill edge-for-image
	scp edge-for-image $(HOST):$(REMOTE_DIR)/
	ssh $(HOST) "cd $(REMOTE_DIR) && ./start-edge-local.sh"


.PHONY: clean
clean:
	rm -f $(BUILD)
