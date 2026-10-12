load("starcm", "download")

download(
    label   = "wrong hash",
    url     = "http://localhost:8000/tests/fixtures/download/payload.txt",
    save_to = "downloaded.txt",
    sha256  = "0000000000000000000000000000000000000000000000000000000000000000",
)
