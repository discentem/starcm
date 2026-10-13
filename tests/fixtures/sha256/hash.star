load("starcm", "sha256")

# The same implementation download() checks against, so this can be pasted
# into download(sha256 = ...).
r = sha256("payload.txt", label = "hash payload")
print("success=%s changed=%s value=%s" % (r.success, r.changed, r.value))
