load("loading", "load_dynamic")
load("download", "download")

f = "x.star"

download(
    name = "download_star",
    url = "http://[::]:8000/examples/only_if.star",
    save_to = f
)

load_dynamic(f)