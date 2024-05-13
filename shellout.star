'''
'''
load("shellout", "exec")

def curlGoogle(args, **kwargs):
    return exec(
        name = "curlGoogle",
        cmd = "curl",
        args = ["google.com"],
        after = lambda args,**kwargs : (print("curling google.com..."))
    )

def cmd(**kwargs):
    # return exec("ping", ["google.com", "-c", "5"])
    return exec(
        name = "echo_hello_after_curlGoogle",
        cmd  = "echo", 
        args = ["hello"],
        # not_if = ('a'+'b') == 'ab',
        after = curlGoogle,
    )