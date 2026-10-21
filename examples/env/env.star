load("starcm", "env")

print("region=%s" % env("STARCM_EXAMPLE_REGION", default = "us-east-1"))
print("unset=%s" % env("STARCM_EXAMPLE_UNSET"))
print("empty=[%s]" % env("STARCM_EXAMPLE_EMPTY", default = "fallback"))
