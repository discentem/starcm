load("starcm", "download")

# Served by the python http.server that the scrut CI job runs from the repo root.
URL = "http://localhost:8000/examples/download/edge/payload.txt"
SHA256 = "77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314"

def show(r):
    print("%s: success=%s changed=%s message=%s" % (r.label, r.success, r.changed, r.message))

show(download(label = "first download", url = URL, save_to = "downloaded.txt", sha256 = SHA256))
show(download(label = "already present", url = URL, save_to = "downloaded.txt", sha256 = SHA256))
