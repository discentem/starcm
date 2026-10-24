load("starcm", "download", "vars")

download(
    label   = "folder with the wrong pin",
    url     = "http://localhost:8000/examples/folders/agent/",
    folder  = True,
    sha256  = "0000000000000000000000000000000000000000000000000000000000000000",
    save_to = vars["managed_dir"] + "/agent",
)
