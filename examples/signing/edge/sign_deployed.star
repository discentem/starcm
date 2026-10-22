load("starcm", "sign", "here")
sign(here("payload.txt"), key = here("private/test.pem"), expires = "30d", label = "deployed")
