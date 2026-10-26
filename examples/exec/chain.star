load("starcm", "exec")

check = exec(label = "is the tool installed?", cmd = "sh", args = ["-c", "command -v not-a-real-tool"])
exec(
    label   = "install the tool",
    cmd     = "echo",
    args    = ["installing not-a-real-tool"],
    not_if  = check,
    live_output = True,
)
