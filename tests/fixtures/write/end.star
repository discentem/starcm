load("starcm", "write")

write("no newline", end = "", label = "positional + end")
write(" -> then newline", label = "default end")
write(str = "str + end", end = "!\n", label = "str + end")
