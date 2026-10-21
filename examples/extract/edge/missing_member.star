load("starcm", "extract")

extract(
    label   = "unpack missing",
    archive = "../files/tool-1.0.zip",
    dest    = "out-missing",
    members = ["tool-1.0/tool", "tool-1.0/nope"],
)
