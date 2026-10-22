load("starcm", "here")

def bundled_banner():
    # here() resolves next to this file, wherever it's called from.
    return here("banner.jinja")
