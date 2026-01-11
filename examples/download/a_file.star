load("starcm", "download")

print(download(
    label = "Downloading Ghostty 1.2.3",
    url = "https://release.files.ghostty.org/1.2.3/Ghostty.dmg",
    save_to = "Ghostty-1.2.3.dmg",
    sha256 = "f35ee91f116e28027ab9f8def45098c7575b44b407ff883a2dcd2985c483206b",
    live_progress = True
))