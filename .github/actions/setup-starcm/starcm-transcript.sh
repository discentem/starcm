#!/usr/bin/env bash
# Installed on PATH as `starcm` in CI. Runs the real starcm ($STARCM_REAL) and
# appends the command, its output and its exit code to $STARCM_TRANSCRIPT, so
# the job log shows what starcm printed even when every scrut test passes.
#
# Output is captured to files and replayed after starcm exits: there are no
# background tee processes to race with the next command, and this works with
# bash 3.2 (macOS). When stdout and stderr are the same pipe (a scrut block
# with output_stream: combined), one file keeps the two streams in order.
# Git Bash can't tell that they're the same pipe, so CI doesn't use this on
# Windows.
set -u

real=$STARCM_REAL

out=$(mktemp)
err=$(mktemp)
trap 'rm -f "$out" "$err"' EXIT

if [ /dev/fd/1 -ef /dev/fd/2 ]; then
  "$real" "$@" >"$out" 2>&1
  rc=$?
  cat "$out"
else
  "$real" "$@" >"$out" 2>"$err"
  rc=$?
  cat "$out"
  cat "$err" >&2
fi

{
  printf '$ starcm'
  printf ' %q' "$@"
  printf '    # %s\n' "${SCRUT_TEST:-$PWD}"
  cat "$out" "$err"
  if [ "$rc" -ne 0 ]; then
    printf '[exit %d]\n' "$rc"
  fi
  printf '\n'
} >>"$STARCM_TRANSCRIPT"

exit "$rc"
