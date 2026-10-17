load("starcm", "exec")

exec(
    label       = "long running installer",
    cmd         = "sh",
    args        = ["-c", "echo step 1; echo step 2"],
    live_output = True,
)
