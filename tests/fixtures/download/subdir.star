load("starcm", "download")

r = download(
    label   = "into a new directory",
    url     = "http://localhost:8000/tests/fixtures/download/payload.txt",
    save_to = "nested/dir/payload.txt",
    sha256  = "77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314",
)
print("success=%s changed=%s" % (r.success, r.changed))
