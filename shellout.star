'''
'''
load("shellout", "exec")

def curlGoogle(args, **kwargs):
    return exec(
        cmd = "curl",
        args = ["google.com"],
        after = lambda args,**kwargs : (print("curling google.com..."))
    )

def cmd():
    # return exec("ping", ["google.com", "-c", "5"])
    return exec(
        cmd  = "echo", 
        args = ["hello"],
        # not_if = ('a'+'b') == 'ab',
        after = curlGoogle,
    )