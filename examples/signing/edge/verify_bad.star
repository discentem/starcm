load("starcm", "here", "verify_all")

verify_all([here("payload.txt"), here("tampered.txt"), here("unsigned.txt")])
