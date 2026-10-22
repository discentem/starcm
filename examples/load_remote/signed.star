load("starcm", "load_remote")

# No sha256 here: the signature, checked against --trusted-keys, says which
# sha256 the module must have.
settings = load_remote(
    url       = "http://localhost:8000/examples/load_remote/remote/screen_lock.star",
    signature = "http://localhost:8000/examples/load_remote/remote/screen_lock.star.sig",
    label     = "screen lock settings",
).value
settings["configure"]()
