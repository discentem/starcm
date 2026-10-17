load("starcm", "download", "load_dynamic", "vars")

download(
    label   = "fetch agent",
    url     = vars["base_url"] + "/agent.pkg",
    save_to = "agent.pkg",
    sha256  = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)
load_dynamic("roles/web.star", label = "web role")
load_dynamic("roles/%s.star" % vars["role"], label = "chosen role")
