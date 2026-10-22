load("starcm", "write")

SCREEN_LOCK_SECONDS = 300

def configure():
    write("require a password %d seconds after the screen saver starts" % SCREEN_LOCK_SECONDS, label = "screen lock")
