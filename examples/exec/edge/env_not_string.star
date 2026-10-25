load("starcm", "exec")

exec(label = "bad env", cmd = "sh", args = ["-c", "true"], env = {"CGO_ENABLED": 0})
