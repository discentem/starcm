load("starcm", "here", "load_remote")

r = load_remote("http://localhost:8000/examples/load_remote/edge/render.star", sha256 = "fb8592c5b4d09d832ed6a84eb3e010c378b32141cf5e1d9913363dc9e86dff2e", label = "renderer")
r.value["render"](here("motd.txt"))
