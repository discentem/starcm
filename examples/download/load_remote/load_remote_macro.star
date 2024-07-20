load("starcm", "load_dynamic")
load("starcm", "download")

def load_remote(url, sha256):
    fname = url.split("/")[-1]
    d = download(
        name = "downloading {}".format(fname),
        url = url,
        save_to = fname,
        sha256 = sha256
    )
    load_dynamic(fname, absolute_path=True)