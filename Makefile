PACKAGE=github.com/tappoy/objectstorage

WORKING_DIRS=tmp
SRC=$(shell find . -name "*.go")
BIN=tmp/$(shell basename $(CURDIR))
FMT=tmp/fmt
TEST=tmp/cover
DOC=Document.txt
COVER=tmp/cover
COVER0=tmp/cover0

.PHONY: all clean cover test lint fmt

all: $(WORKING_DIRS) fmt $(BIN) test $(DOC)

clean:
	rm -rf $(WORKING_DIRS)

$(WORKING_DIRS):
	mkdir -p $(WORKING_DIRS)

fmt: $(SRC)
	go fmt ./...

go.sum: go.mod
	go mod tidy

$(BIN): $(SRC) go.sum
	go build -o $(BIN)

$(DOC): $(SRC)
	go doc -all . > $(DOC)

cover: $(COVER)
	grep "0$$" $(COVER) | sed 's!$(PACKAGE)!.!' | tee $(COVER0)

test: $(BIN)
	go test -v -tags=mock -vet=all -cover -coverprofile=$(COVER)

lint: $(BIN)
	go vet
