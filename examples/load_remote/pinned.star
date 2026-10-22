load("starcm", "load_remote")

r = load_remote(
    "http://localhost:8000/examples/load_remote/remote/screen_lock.star",
    sha256 = "3ad71837f20bade1d1eb37f5552d173bd4e03551eed67a829a2cd2173640d983",
    label  = "screen lock settings",
)
print("changed=%s" % r.changed)
r.value["configure"]()
