load("starcm", "download")

# Without save_to, download() saves into the store (--store-dir) and returns
# the stored file's path as value.
URL = "http://localhost:8000/examples/download/edge/payload.txt"
SHA256 = "77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314"

def show(r):
    print("%s: changed=%s value=%s" % (r.label, r.changed, r.value.replace("\\", "/")))

show(download(label = "into the store", url = URL, sha256 = SHA256))
show(download(label = "already stored", url = URL, sha256 = SHA256))
show(download(label = "named", url = URL, sha256 = SHA256, file_name = "renamed.txt"))
