load("starcm", "download", "install_mac_pkg")

# dockutil 3.1.3. download() checks the sha256 and keeps the package in
# starcm's local store.
pkg = download(
    label  = "fetch dockutil",
    url    = "https://github.com/kcrawford/dockutil/releases/download/3.1.3/dockutil-3.1.3.pkg",
    sha256 = "f60db8273fe80d7c4824588bedb538a5387675c3917edb0010dab9f52d3f2582",
)

# Install it only if dockutil's developer signed it and Apple notarized it.
r = install_mac_pkg(
    pkg,
    signer  = "Developer ID Installer: Kyle Crawford (Z5J8CJBUWC)",
    id      = "dockutil.cli.tool",
    version = "3.1.3",
    label   = "install dockutil",
)
if r.changed:
    print("installed dockutil, signed by %s" % r.value.signer)
else:
    print(r.message)
