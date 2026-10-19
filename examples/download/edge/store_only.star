load("starcm", "download")

download(
    label   = "to a path",
    url     = "http://localhost:8000/examples/download/edge/payload.txt",
    save_to = "downloaded.txt",
    sha256  = "77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314",
)
