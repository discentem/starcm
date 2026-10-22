load("starcm", "load_remote")

# The same module twice in one run: it runs once, and both calls get its globals.
for i in [1, 2]:
    r = load_remote("http://localhost:8000/examples/load_remote/edge/hello.star", sha256 = "d5060e7e366df6090f58f576d6c499df6e44ba7d8b334e68a76dcf1d02b32b03", label = "hello %d" % i)
    print("%d: %s" % (i, r.value["GREETING"]))
