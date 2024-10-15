build:
	bazel build //:starcm

run_if_statements:
	bazel run :starcm -- --root_file examples/if_statements/if_statements.star
# Updates MODULE.bazel file with top level external dependencies
tidy:
	bazel mod tidy

gazelle:
	bazel run //:gazelle