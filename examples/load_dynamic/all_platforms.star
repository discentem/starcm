load("starcm", "load_dynamic")

for platform in ["macos", "windows"]:
    settings = load_dynamic(module = "platforms/%s.star" % platform, label = "load " + platform).value
    print("%s locks the screen after %d seconds" % (platform, settings["SCREEN_LOCK_SECONDS"]))
