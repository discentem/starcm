load("starcm", "template")

# By default, template and destination are relative to this file's directory...
template(label = "next to me", template = "hello.tpl", destination = "here.txt", data = {"where": "sub"})

# ...an explicit working_directory is used as given instead.
template(
    label             = "from the current directory",
    template          = "sub/hello.tpl",
    destination       = "there.txt",
    data              = {"where": "the current directory"},
    working_directory = ".",
)
