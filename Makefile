# Updates MODULE.bazel file with top level external dependencies

deps: tidy gazelle

tidy:
	bazel mod tidy

gazelle:
	bazel run :gazelle
