load("starcm", "write")

write("hello from write", label = "greet")
write(str = "keyword form", label = "keyword")
write("no newline -> ", end = "", label = "partial")
write("then newline", label = "rest")
write(["lists", "print", "as", "Starlark"], label = "list")
r = write("written values are returned", label = "returned")
print("value=%s success=%s changed=%s" % (r.value, r.success, r.changed))
