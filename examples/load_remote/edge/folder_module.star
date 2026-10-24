load("starcm", "load_remote")

load_remote("http://localhost:8000/examples/folders/agent/", folder = True, module = "setup.star", sha256 = "f7e8fbe643ea525dce9c8e0138f2ff4cad6d5ab1d539ed7a35ac55be380981ca", label = "no such module")
