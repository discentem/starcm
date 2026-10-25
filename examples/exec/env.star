load("starcm", "exec")

# env adds to the environment starcm runs with, so PATH and HOME are still
# there. A variable starcm already has, like GREETING below, is overridden.
exec(
    label       = "build",
    cmd         = "sh",
    args        = ["-c", "echo \"GOFLAGS=$GOFLAGS CGO_ENABLED=$CGO_ENABLED GREETING=$GREETING\"; command -v sh >/dev/null && echo 'PATH still finds sh'"],
    env         = {"GOFLAGS": "-mod=readonly", "CGO_ENABLED": "0", "GREETING": "from env"},
    live_output = True,
)
