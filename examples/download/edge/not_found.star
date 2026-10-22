load("starcm", "download", "vars")

download(
    label   = "404",
    url     = "http://localhost:8000/examples/download/edge/does-not-exist.txt",
    save_to = vars["managed_dir"] + "/nope.txt",
    sha256  = "0000000000000000000000000000000000000000000000000000000000000000",
)
