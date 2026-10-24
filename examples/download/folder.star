load("starcm", "download", "vars")

# folder = True fetches a folder. sha256 pins its manifest,
# starcm-manifest.json, which lists every file in the folder with its sha256.
r = download(
    label   = "fetch agent files",
    url     = "http://localhost:8000/examples/folders/agent/",
    folder  = True,
    sha256  = "f7e8fbe643ea525dce9c8e0138f2ff4cad6d5ab1d539ed7a35ac55be380981ca",
    save_to = vars["managed_dir"] + "/agent",
)
print(r.message)
