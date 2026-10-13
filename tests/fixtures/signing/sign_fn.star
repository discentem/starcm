load("starcm", "sign")

def show(r):
    print("%s: changed=%s value=%s" % (r.label, r.changed, r.value))

show(sign(path = "payload.txt", key = "private/untrusted.pem", key_id = "other", label = "other key"))
show(sign("payload.txt", key = "private/test.pem", label = "test key"))
show(sign("payload.txt", key = "private/test.pem", label = "unchanged"))
