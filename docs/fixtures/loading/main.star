load("starcm", "write", say = "write")
load("lib/users.star", "ADMINS", "greet")

for admin in ADMINS:
    greet(admin)
say("aliases work too", label = "alias")
