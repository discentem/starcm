load("starcm", "write")

# Strings print as plain text, like print(): no quotes, escapes applied.
write("hello world", label = "string")
write('she said "hi"', label = "embedded quotes")
write("tab\there", label = "tab")

# Other values print their Starlark representation.
write(42, label = "int")
write(["a", "b"], label = "list")
write({"k": "v"}, label = "dict")
write(None, label = "none")
