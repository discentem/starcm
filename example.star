'''
'''
load("shellout", "exec")
load("shellout.star", "cmd")
load("struct", "struct")


# import a function defined in .star file
print(cmd())

thing = struct(a=1, b=2)

print(thing.a)

exec(
    name = "echo hello :D",
    cmd  = "echo", 
    args = ["hello"],
)
