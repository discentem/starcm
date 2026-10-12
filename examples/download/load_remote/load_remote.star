load("load_remote_macro.star", "load_remote")

# use the load_remote macro to download and load the remote file.
# Served by `python3 -m http.server 8000` from the repo root.
load_remote(
    url = "http://localhost:8000/examples/nested_files/e.star",
    sha256 = "6a6afe8b43b89e345842f74d84cf92e280862d648f01f8f0fa356a9d4c6e8796"
)
