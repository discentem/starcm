'''
'''
load("shellout", "exec")

def curlGoogle(args, **kwargs):
    return exec(
        name = "curlGoogle",
        cmd = "curl",
        args = ["http://www.google.com", "-w", "%{http_code}", "-o", "/dev/null", "-I"],
    )

def cmd(args=None, **kwargs):
    res = curlGoogle(args, **kwargs)
    print(res)
    return exec(
        name = "echo_hello_after_curlGoogle",
        cmd  = "echo", 
        args = ["hello googs"],
        # not_if = ('a'+'b') == 'ab',
    )

print(cmd())