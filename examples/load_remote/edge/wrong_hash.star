load("starcm", "load_remote")

load_remote("http://localhost:8000/examples/load_remote/edge/hello.star", sha256 = "0000000000000000000000000000000000000000000000000000000000000000", label = "wrong hash")
