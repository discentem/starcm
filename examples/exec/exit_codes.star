load("starcm", "exec")

ok = exec(label = "grep finds nothing", cmd = "sh", args = ["-c", "exit 1"], expected_exit_code = 1)
print("expected 1: success=%s error=%s" % (ok.success, ok.error))

bad = exec(label = "unexpected failure", cmd = "sh", args = ["-c", "echo oops >&2; exit 3"])
print("unexpected: success=%s error=%s value=%r" % (bad.success, bad.error, bad.value))
if not bad.success:
    print("handle the failure here, or stop the run with fail()")
