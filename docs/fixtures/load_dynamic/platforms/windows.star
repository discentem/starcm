load("starcm", "write")

SCREEN_LOCK_SECONDS = 600

def configure():
    write("Windows: lock the screen after %d seconds idle" % SCREEN_LOCK_SECONDS, label = "Windows screen lock")
