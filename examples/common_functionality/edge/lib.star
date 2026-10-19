load("starcm", "exec")

def call_with_timeout():
    exec(label = "lib call with timeout", cmd = "true", args = [], timeout = "100ms")
