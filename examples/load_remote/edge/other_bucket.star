load("starcm", "load_remote")

load_remote("s3://config/main.star", signature = "s3://elsewhere/main.star.sig", label = "other bucket")
