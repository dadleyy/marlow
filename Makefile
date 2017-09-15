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

all: $(EXE)

$(EXE): $(VENDOR_DIR) $(GO_SRC) $(LIB_SRC) $(LINT_RESULT)
	$(COMPILE) $(BUILD_FLAGS) -o $(EXE) $(MAIN)

$(LINT_RESULT): $(GO_SRC)
	$(LINT) $(LINT_FLAGS) $(shell go list $(SRC_DIR)/... | grep -v 'interchange')
	touch $(LINT_RESULT)

test: $(GO_SRC) $(LINT_RESULT) $(COVERAGE_REPORT)
	$(GO) vet $(shell go list ./... | grep -vi 'vendor\|testing')

$(COVERAGE_REPORT):
	$(COVERAGE) -v -parallel=1 -covermode=atomic -coverprofile=$(COVERAGE_REPORT) $(SRC_DIR)/...

$(VENDOR_DIR):
	go get -u github.com/Masterminds/glide
	go get -u github.com/golang/lint/golint
	go get -u github.com/haya14busa/goverage
	$(GLIDE) install

clean:
	rm -rf $(COVERAGE_REPORT)
	rm -rf $(LINT_RESULT)
	rm -rf $(VENDOR_DIR)
	rm -rf $(EXE)
	rm -rf $(EXAMPLE_OBJS)
