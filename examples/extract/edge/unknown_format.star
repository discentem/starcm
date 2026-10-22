load("starcm", "extract", "here", "vars")

extract(label = "unpack rar", archive = here("tool.rar"), dest = vars["managed_dir"] + "/out-rar")
