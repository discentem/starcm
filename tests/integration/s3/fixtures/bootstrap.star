load("starcm", "load_remote", "vars")

# The only configuration on the machine to start with. load_remote() fetches
# main.star from s3://config into the local store, pinned by its signature,
# and runs it from there: its .sig is stored next to it for --trusted-keys.
load_remote(
    label     = "run main.star",
    url       = "s3://config/main.star",
    signature = "s3://config/main.star.sig",
    endpoint  = vars["s3_endpoint"],
)
