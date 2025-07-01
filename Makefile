STARCM_BIN := $(shell bazel cquery :starcm --output=files 2>/dev/null)

build: deps
	bazel build :starcm

install: build
	sudo cp $(STARCM_BIN) /usr/local/bin/starcm

deps: tidy gazelle

tidy:
	bazel mod tidy

gazelle:
	bazel run :gazelle


