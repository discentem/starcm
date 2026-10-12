load("starcm", "write")

write("only_if True runs", only_if = True, label = "only_if true")
write("only_if False is skipped", only_if = False, label = "only_if false")
write("not_if False runs", not_if = False, label = "not_if false")
write("not_if True is skipped", not_if = True, label = "not_if true")
