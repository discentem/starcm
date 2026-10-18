load("starcm", "download", "install_mac_pkg", "vars")

# Installomator 10.9: Developer ID signed by Armin Briegel (JME5BW3F3R),
# notarized, installs /usr/local/Installomator/Installomator.sh.
SHA256 = "8974888d6b27071cd6418d2d5afc53e39f7dff682641e7faed64d3d379679fad"

download(
    label   = "fetch Installomator",
    url     = "https://github.com/Installomator/Installomator/releases/download/v10.9/Installomator-10.9.pkg",
    save_to = "Installomator-10.9.pkg",
    sha256  = SHA256,
)

r = install_mac_pkg(
    "Installomator-10.9.pkg",
    signer  = vars.get("signer", "Developer ID Installer: Armin Briegel (JME5BW3F3R)"),
    sha256  = vars.get("pkg_sha256", SHA256),
    id      = "com.scriptingosx.Installomator",
    version = "10.9",
    label   = "install Installomator",
)
print("%s | changed=%s" % (r.message, r.changed))
if r.value:
    print("team=%s notarized=%s" % (r.value.team_id, r.value.notarized))
