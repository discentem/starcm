load("starcm", "template")

def render(label, name, **kwargs):
    template(
        label       = label,
        template    = "hello.tpl",
        data        = {"name": name, "age": 1},
        destination = "out.txt",
        **kwargs
    )

render("elsewhere", "sub", working_directory = "sub")
render("here", "top")
