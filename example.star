'''
'''
load("shellout", "exec")
load("shellout.star", "cmd")
load("struct", "struct")


# # import a function defined in .star file
print(cmd())

# exec(
#     name = "echo hello :D",
#     cmd  = "echo", 
#     args = ["hello"],
# )
