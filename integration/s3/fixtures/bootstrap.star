load("starcm", "load_dynamic", "s3_download", "vars")

# The only configuration on the machine to start with. main.star comes from
# s3://fleet into the local store, pinned by its signature, and runs from
# there: its .sig is stored next to it for --trusted-keys.
main = s3_download(
    label     = "fetch main.star",
    bucket    = "fleet",
    key       = "main.star",
    signature = "main.star.sig",
    endpoint  = vars["s3_endpoint"],
).value

load_dynamic(main, label = "run main.star")
