'''
'''
load("shellout", "exec")

def msg(s):
    def ret(args, **kwargs):
        return s
    return ret

def curlGoogle(args, **kwargs):
    return exec(
        name = "curlGoogle",
        cmd = "curl",
        args = ["google.com"],
        before = msg("curling google.com...")
    )

def cmd(**kwargs):
    # return exec("ping", ["google.com", "-c", "5"])
    return exec(
        name = "echo_hello_after_curlGoogle",
        cmd  = "echo", 
        args = ["hello"],
        # not_if = ('a'+'b') == 'ab',
        before = curlGoogle,
    )