load("starcm", "download", "exec", "extract", "file", "platform", "temp_dir", "vars")

# Terraform 1.17.0 for Apple silicon: Developer ID signed by HashiCorp
# (D38WU7D763) and notarized. The config shrinks to download -> extract ->
# file(source, signer); file() checks the copy it installs.
p = platform()
if (p.os, p.arch) != ("darwin", "arm64"):
    fail("this test needs an Apple silicon Mac, not %s/%s" % (p.os, p.arch))

zip = download(
    label  = "fetch terraform",
    url    = "https://releases.hashicorp.com/terraform/1.17.0/terraform_1.17.0_darwin_arm64.zip",
    sha256 = "7472a54cca911d46c0e399a5ebbce44444912d0d4dad7495f37f1fa51e7ca6b7",
)

unpacked = temp_dir("starcm-terraform-test")
extract(zip.value, dest = unpacked, members = ["terraform"], label = "unpack terraform")
binary = unpacked + "/terraform"

if vars.get("tamper") == "yes":
    # Something changes the binary between extract() and file()...
    exec(label = "tamper", cmd = "sh", args = ["-c", "printf junk >> \"$1\"", "sh", binary], check = True)

r = file(
    label  = "install terraform",
    path   = vars["bin_dir"] + "/terraform",
    source = binary,
    signer = vars.get("signer", "Developer ID Application: Hashicorp, Inc. (D38WU7D763)"),
    mode   = 0o755,
)
print("changed=%s signer=%s notarized=%s" % (r.changed, r.value.signer, r.value.notarized))
