load("load_remote_macro.star", "load_remote")

# use the load_remote macro to download and load the remote file
load_remote(
    url = "http://localhost:8080/only_if/only_if.star",
    sha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)