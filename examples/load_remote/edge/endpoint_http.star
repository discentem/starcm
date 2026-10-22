load("starcm", "load_remote")

load_remote("http://localhost:8000/examples/load_remote/edge/hello.star", sha256 = "d5060e7e366df6090f58f576d6c499df6e44ba7d8b334e68a76dcf1d02b32b03", endpoint = "http://127.0.0.1:9000", label = "endpoint")
