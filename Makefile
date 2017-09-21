GO=go

COMPILE=$(GO) build
LDFLAGS="-s -w"
BUILD_FLAGS=-x -v -ldflags $(LDFLAGS)

GLIDE=glide
VENDOR_DIR=vendor

LINT=golint
LINT_FLAGS=-set_exit_status

EXE=mc
MAIN=$(wildcard ./marlowc/main.go)

COVERAGE=goverage
COVERAGE_REPORT=coverage.out

LIB_DIR=./marlow
SRC_DIR=./marlowc
EXAMPLE_DIR=./examples
EXAMPLE_MODEL_DIR=$(EXAMPLE_DIR)/library/models

CYCLO=gocyclo
CYCLO_FLAGS=-over 15

MISSPELL=misspell

LIB_SRC=$(wildcard $(LIB_DIR)/**/*.go $(LIB_DIR)/*.go)
GO_SRC=$(wildcard $(MAIN) $(SRC_DIR)/**/*.go $(SRC_DIR)/*.go)

EXAMPLE_EXE=$(EXAMPLE_DIR)/lib
EXAMPLE_MAIN=$(wildcard $(EXAMPLE_DIR)/library/main.go)
EXAMPLE_SRC=$(wildcard $(EXAMPLE_DIR)/library/**/*.go)
EXAMPLE_OBJS=$(patsubst %.go,%.marlow.go,$(EXAMPLE_SRC))

VET=$(GO) vet
VET_FLAGS=

MAX_TEST_CONCURRENCY=10

TEST_FLAGS=-covermode=atomic -coverprofile={{.Dir}}/.coverprofile
TEST_LIST_FMT='{{if len .TestGoFiles}}"go test {{.ImportPath}} $(TEST_FLAGS)"{{end}}'

EXAMPLE_TEST_FLAGS=-covermode=atomic

all: $(EXE)

$(EXE): $(VENDOR_DIR) $(GO_SRC) $(LIB_SRC)
	$(COMPILE) $(BUILD_FLAGS) -o $(EXE) $(MAIN)

lint: $(GO_SRC)
	$(LINT) $(LINT_FLAGS) $(LIB_DIR)/...
	$(LINT) $(LINT_FLAGS) $(MAIN)

test: $(GO_SRC) $(VENDOR_DIR) $(INTERCHANGE_OBJ) lint
	$(VET) $(VET_FLAGS) $(SRC_DIR)
	$(VET) $(VET_FLAGS) $(LIB_DIR)
	$(VET) $(VET_FLAGS) $(MAIN)
	$(CYCLO) $(CYCLO_FLAGS) $(LIB_SRC)
	$(MISSPELL) -error $(LIB_SRC) $(MAIN)
	$(GO) list -f $(TEST_LIST_FMT) $(LIB_DIR)/... | xargs -L 1 sh -c

test-example: $(EXAMPLE_SRC)
	$(GO) run $(MAIN) -input=$(EXAMPLE_MODEL_DIR)
	$(GO) test $(EXAMPLE_TEST_FLAGS) -p=$(MAX_TEST_CONCURRENCY) $(EXAMPLE_MODEL_DIR)

$(VENDOR_DIR):
	go get -u github.com/client9/misspell/cmd/misspell
	go get -u github.com/fzipp/gocyclo
	go get -u github.com/Masterminds/glide
	go get -u github.com/golang/lint/golint
	$(GLIDE) install

example: $(EXAMPLE_SRC) $(EXAMPLE_MAIN)
	$(GO) run $(MAIN) -input=$(EXAMPLE_MODEL_DIR)
	$(COMPILE) $(BUILD_FLAGS) -o $(EXAMPLE_EXE) $(EXAMPLE_MAIN)

clean-example:
	rm -rf $(EXAMPLE_OBJS)
	rm -rf $(EXAMPLE_EXE)

clean: clean-example
	rm -rf $(COVERAGE_REPORT)
	rm -rf $(LINT_RESULT)
	rm -rf $(VENDOR_DIR)
	rm -rf $(EXE)
