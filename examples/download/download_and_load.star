load("loading", "load_dynamic")
load("download", "download")
load("write", "write")

f = "z.star"

d = download(
    name = "download_star",
    url = "http://localhost:8080/only_if/only_if.star",
    save_to = f,
    sha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

write(str=d.output)

load_dynamic(f, absolute_path=True)