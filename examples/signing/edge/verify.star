load("starcm", "verify", "here")

def show(r):
    print("%s: success=%s value=%s error=%s" % (r.label, r.success, r.value, r.error))

show(verify(here("payload.txt"), label = "signed"))
show(verify(path = here("copy.txt"), signature = here("payload.txt.sig"), label = "signature= kwarg"))
show(verify(here("tampered.txt"), signature = here("payload.txt.sig"), label = "tampered"))
show(verify(here("unsigned.txt"), label = "no signature"))
