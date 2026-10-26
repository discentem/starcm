load("starcm", "here", "verify_all")

v = verify_all([here("payload.txt")])[0]
print("signed: key_id=%s release=%r" % (v.key_id, v.release))
