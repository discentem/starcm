load("starcm", "glob", "here")

glob(here("../third_party/*/[.mod"), label = "bad pattern")
