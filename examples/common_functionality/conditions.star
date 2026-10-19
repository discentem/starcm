load("starcm", "exec", "write")

installed = exec(label = "already installed?", cmd = "sh", args = ["-c", "exit 1"])

write("installing", label = "install", not_if = installed.success)
write("configuring", label = "configure", only_if = installed.success)
skipped = write("never printed", label = "skipped", only_if = False)
print("skipped: %s" % skipped)
