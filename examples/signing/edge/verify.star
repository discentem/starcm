load("starcm", "verify", "here")

def show(name, v):
    print("%s: verified=%s key_id=%s error=%s" % (name, v.verified, v.key_id, v.error))

show("signed", verify(here("payload.txt")))
show("signature= kwarg", verify(path = here("copy.txt"), signature = here("payload.txt.sig")))
show("tampered", verify(here("tampered.txt"), signature = here("payload.txt.sig")))
show("no signature", verify(here("unsigned.txt")))
