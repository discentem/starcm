load("starcm", "path")

print(path.join("/opt/agent", "bin", "agent"))
print(path.join("/opt/agent/", "../tools", ""))   # cleaned, like Go's filepath.Join
