'''
'''
load("shellout", "exec")

def curlGoogle(args, **kwargs):
    return exec(
        name = "curlGoogle",
        cmd = "curl",
        args = ["google.com"],
    )

def cmd(args=None, **kwargs):
    res = curlGoogle(args, **kwargs)
    print(res)
    # return exec("ping", ["google.com", "-c", "5"])
    return exec(
        name = "echo_hello_after_curlGoogle",
        cmd  = "echo", 
        args = ["hello googs"],
        # not_if = ('a'+'b') == 'ab',
    )