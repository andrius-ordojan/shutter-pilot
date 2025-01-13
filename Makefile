OUTPUT_DIR := dist

ARCH := amd64

.PHONY: all
all: build-linux build-windows build-darwin

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=$(ARCH) go build -o $(OUTPUT_DIR)/linux/shutter-pilot .

.PHONY: build-windows
build-windows:
	GOOS=windows GOARCH=$(ARCH) go build -o $(OUTPUT_DIR)/win/shutter-pilot.exe .

.PHONY: build-darwin
build-darwin:
	GOOS=darwin GOARCH=$(ARCH) go build -o $(OUTPUT_DIR)/darwin/shutter-pilot .

.PHONY: clean
clean:
	rm -r $(OUTPUT_DIR)

.PHONY: test
test:
	go test

.PHONY: clear_tmp
clear_tmp:
	rm -r tmp*
