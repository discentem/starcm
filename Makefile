# Updates MODULE.bazel file with top level external dependencies
tidy:
	bazel mod tidy

gazelle:
	bazel run :gazelle
