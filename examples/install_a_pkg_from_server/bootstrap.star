load("starcm", "download", "load_dynamic")

save_path = "/tmp/install_json_pkg.star"

res = download(
    name = "download install_json_pkg.star",
    url = "http://localhost:8000/examples/install_a_pkg_from_server/install_json_pkg.star",
    sha256 = "37fb875a1424d065c394beb97186db4a585955dafd85f1668c2151dd9ce931c6",
    save_to = save_path,
)

if res.success:
    # load the downloaded file dynamically
    load_dynamic(save_path, absolute_path = True)
else:
    print("Failed to download install_go.star: %s" % res.error)