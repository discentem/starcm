load("starcm", "download")

r = download(
    label  = "fetch agent files into the store",
    url    = "http://localhost:8000/examples/folders/agent/",
    sha256 = "f7e8fbe643ea525dce9c8e0138f2ff4cad6d5ab1d539ed7a35ac55be380981ca",
)
print("changed=%s" % r.changed)
print(r.value.replace("\\", "/"))  # / on every platform, for this example
