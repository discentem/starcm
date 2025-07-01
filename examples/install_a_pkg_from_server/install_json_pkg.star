# load functions from the starcm module/stdlb that ships inside of starcm
load("starcm", "download", "exec", "write")

# define a runtime function for install a package on macOS
def install_pkg(filepath):
    # call exec() from starcm module
    return exec(
        name = "install %s" % filepath,
        cmd = "/usr/bin/sudo",
        # include -verboseR for bigger packages to see more details
        args = ["/usr/sbin/installer", "-pkg", filepath, "-target", "/"],
        live_output = False,
    )
# call download() from starcm module to download go from the internet
download_go_result = download(
    name = "download examplejson-1.0.pkg",
    url = "http://localhost:8000/examples/install_a_pkg_from_server/build/examplejson-1.0.pkg",
    sha256 = "5097954f0a80939a0b88d8d0f1bfce2143c3c7a1ab07ea0e5774f1f6dda1c47e",
    save_to = "examplejson-1.0.pkg",
)

print(download_go_result)

# call the function we defined above to install the downloaded package
install_go_result = install_pkg(
    filepath = "examplejson-1.0.pkg",
)

print("""
{
    "name": "%s",
    "changed": %s,
    "success": %s
}

""" % (
    install_go_result.name,
    install_go_result.changed,
    install_go_result.success
))
