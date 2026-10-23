load("starcm", "install_remote_mac_pkg")

install_remote_mac_pkg(
    "http://localhost:8000/examples/install_remote_mac_pkg/remote/unsigned.pkg",
    sha256 = "4facc51893659121661fc6337f2b63ce5abf64b24ba7a73f399d656eb6d71edf",
    signer = "Developer ID Installer: Example Corp (ABCDE12345)",
    label  = "unsigned",
)
print("never reached")
