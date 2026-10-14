load("starcm", "download", "exec")

r = download(
    label  = "into the store",
    url    = "http://localhost:8000/tests/fixtures/download/payload.txt",
    sha256 = "77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314",
)

def cat(label):
    return exec(label = label, cmd = "sh", args = ["-c", "cat \"$1\"", "sh", r.value])

print("read: %r" % cat("read").value)

# Change the stored file between download() and the next exec().
exec(label = "tamper", cmd = "sh", args = ["-c", "chmod u+w \"$1\" && echo edited >> \"$1\"", "sh", r.value])

cat("read again")
