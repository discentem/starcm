load("starcm", "download", "load_dynamic")

save_path = "/tmp/install_json_pkg.star"

res = download(
    name = "download install_json_pkg.star",
    url = "http://localhost:8000/examples/install_a_pkg_from_server/install_json_pkg.star",
    sha256 = "83c08fe827e9a9eb2240712d5d8d520d8061b24eb579c6a5fe5524bef6f85b5c",
    save_to = save_path,
)

if res.success:
    # load the downloaded file dynamically
    load_dynamic(save_path, absolute_path = True)
else:
    print("Failed to download install_go.star: %s" % res.error)