load("starcm", "extract", "here", "vars")

extract(
    label   = "unpack missing",
    archive = here("../files/tool-1.0.zip"),
    dest    = vars["managed_dir"] + "/out-missing",
    members = ["tool-1.0/tool", "tool-1.0/nope"],
)
