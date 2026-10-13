load("starcm", "verify")

def show(r):
    print("%s: success=%s value=%s error=%s" % (r.label, r.success, r.value, r.error))

show(verify("payload.txt", label = "signed"))
show(verify(path = "copy.txt", signature = "payload.txt.sig", label = "signature= kwarg"))
show(verify("tampered.txt", signature = "payload.txt.sig", label = "tampered"))
show(verify("unsigned.txt", label = "no signature"))
