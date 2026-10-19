load("starcm", "download")

download(
    label   = "404",
    url     = "http://localhost:8000/examples/download/edge/does-not-exist.txt",
    save_to = "nope.txt",
    sha256  = "0000000000000000000000000000000000000000000000000000000000000000",
)
