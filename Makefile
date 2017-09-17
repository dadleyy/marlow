GO=go

COMPILE=$(GO) build
BUILD_FLAGS=-x -v

GLIDE=glide
VENDOR_DIR=vendor

LINT=golint
LINT_FLAGS=-set_exit_status
LINT_RESULT=.lint-results

EXE=mc
MAIN=$(wildcard ./marlowc/main.go)

COVERAGE=goverage
COVERAGE_REPORT=coverage.out

LIB_DIR=./marlow
SRC_DIR=./marlowc
EXAMPLE_DIR=./examples

LIB_SRC=$(wildcard $(LIB_DIR)/**/*.go $(LIB_DIR)/*.go)
GO_SRC=$(wildcard $(MAIN) $(SRC_DIR)/**/*.go $(SRC_DIR)/*.go)
EXAMPLE_OBJS=$(wildcard $(EXAMPLE_DIR)/**/*.marlow.go)

VET=$(GO) vet
VET_FLAGS=

MAX_TEST_CONCURRENCY=10
TEST_FLAGS=
TEST_LIST_FMT='{{if len .TestGoFiles}}"go test {{.Name}} $(TEST_FLAGS)"{{end}}'

all: $(EXE)

$(EXE): $(VENDOR_DIR) $(GO_SRC) $(LIB_SRC) $(LINT_RESULT)
	$(COMPILE) $(BUILD_FLAGS) -o $(EXE) $(MAIN)

lint: $(GO_SRC)
	$(LINT) $(LINT_FLAGS) $(LIB_DIR)
	$(LINT) $(LINT_FLAGS) $(MAIN)

test: $(GO_SRC) $(VENDOR_DIR) $(INTERCHANGE_OBJ) lint
	$(VET) $(VET_FLAGS) $(SRC_DIR)
	$(VET) $(VET_FLAGS) $(LIB_DIR)
	$(VET) $(VET_FLAGS) $(MAIN)
	$(GO) test -covermode=atomic -coverprofile=.coverprofile -p=$(MAX_TEST_CONCURRENCY) $(LIB_DIR)

$(VENDOR_DIR):
	go get -u github.com/Masterminds/glide
	go get -u github.com/golang/lint/golint
	$(GLIDE) install

clean:
	rm -rf $(COVERAGE_REPORT)
	rm -rf $(LINT_RESULT)
	rm -rf $(VENDOR_DIR)
	rm -rf $(EXE)
	rm -rf $(EXAMPLE_OBJS)
