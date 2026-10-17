load("starcm", "exec")

# cmd is run directly, not through a shell. Ask for one to use pipes,
# redirects or globs.
r = exec(
    label = "count lines",
    cmd   = "sh",
    args  = ["-c", "printf 'a\\nb\\nc\\n' | wc -l | tr -d ' '"],
)
print("lines: %s" % r.value.strip())
