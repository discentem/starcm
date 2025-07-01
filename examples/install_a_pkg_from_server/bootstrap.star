load("starcm", "download", "load_dynamic")

save_path = "/tmp/install_json_pkg.star"

res = download(
    name = "download install_json_pkg.star",
    url = "http://localhost:8000/examples/install_a_pkg_from_server/install_json_pkg.star",
    sha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    save_to = save_path,
)

if res.success:
    # load the downloaded file dynamically
    load_dynamic(save_path)
else:
    print("Failed to download install_go.star: %s" % res.error)