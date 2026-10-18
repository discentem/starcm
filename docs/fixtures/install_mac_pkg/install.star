load("starcm", "download", "install_mac_pkg")

# From `pkgutil --check-signature` on a copy of the package you trust.
AGENT_SIGNER = "1E 77 DE 6F E4 7D 42 69 84 C5 AA 18 0D 58 1F 32 46 19 7F 73 C0 C7 13 27 7B 84 40 08 B9 36 E0 93"
AGENT_SHA256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08"

pkg = download(
    label  = "fetch agent",
    url    = "https://downloads.example.com/agent-1.2.0.pkg",
    sha256 = AGENT_SHA256,
)

r = install_mac_pkg(
    pkg.value,
    signer_sha256 = AGENT_SIGNER,
    id            = "com.example.agent",
    version       = "1.2.0",
    label         = "install agent",
)
print("%s (changed=%s)" % (r.message, r.changed))
