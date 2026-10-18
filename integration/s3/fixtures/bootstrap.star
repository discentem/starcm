load("starcm", "load_dynamic", "s3_download", "vars")

# The only configuration on the machine to start with. Everything else comes
# from s3://fleet, pinned by sha256 and signed.
for key, sha256 in [("main.star", "c0f6f51a8336ea11bd03840f9399f71c65f0c000338160968537109b76ffb641"), ("main.star.sig", "165622f3a5401fb4126dd688dad6f7538b2fb4617203104d78fde398dc16434f")]:
    s3_download(
        label    = "fetch " + key,
        bucket   = "fleet",
        key      = key,
        sha256   = sha256,
        save_to  = "fleet/" + key,
        endpoint = vars["s3_endpoint"],
    )

load_dynamic("fleet/main.star", label = "run main.star")
