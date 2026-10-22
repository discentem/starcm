load("starcm", "sign", "here")

def show(r):
    print("%s: changed=%s value=%s" % (r.label, r.changed, r.value))

show(sign(path = here("payload.txt"), key = here("private/untrusted.pem"), key_id = "other", expires = "30d", label = "other key"))
show(sign(here("payload.txt"), key = here("private/test.pem"), expires = "2099-01-01", label = "test key"))
show(sign(here("payload.txt"), key = here("private/test.pem"), expires = "2099-01-01", label = "unchanged"))
show(sign(here("payload.txt"), key = here("private/test.pem"), expires = "2099-01-01", release = "2026.10", label = "release"))
