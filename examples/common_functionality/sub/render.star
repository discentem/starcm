load("starcm", "template", "vars")

# template ships next to this file, so the relative "hello.tpl" is found here.
# destination is an output, so it's a full path: here, under the folder the
# run passes as --var managed_dir.
template(label = "next to me", template = "hello.tpl", destination = vars["managed_dir"] + "/hello.txt", data = {"where": "sub"})
