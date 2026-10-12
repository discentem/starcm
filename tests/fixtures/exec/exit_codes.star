load("starcm", "exec")

expected = exec(label = "expected exit", cmd = "sh", args = ["-c", "exit 2"], expected_exit_code = 2)
print("expected: success=%s" % expected.success)

unexpected = exec(label = "unexpected exit", cmd = "sh", args = ["-c", "exit 3"])
print("unexpected: success=%s error=%s" % (unexpected.success, unexpected.error))
