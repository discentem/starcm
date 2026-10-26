load("starcm", "exec", "write")

installed = exec(label = "already installed?", cmd = "sh", args = ["-c", "exit 1"])

# A result in only_if or not_if stands for its success.
write("installing", label = "install", not_if = installed)
write("configuring", label = "configure", only_if = installed)
skipped = write("never printed", label = "skipped", only_if = False)
print("skipped: %s" % skipped)
