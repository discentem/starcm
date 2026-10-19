load("starcm", "template")

# A launchd job that runs the agent's check-in every 15 minutes.
r = template(
    label       = "agent launchd job",
    template    = "agent.plist.tpl",
    destination = "com.example.agent.plist",
    data        = {
        "label":            "com.example.agent",
        "program":          ["/usr/local/bin/agent", "check-in"],
        "interval_seconds": 900,
        "run_at_load":      True,
    },
)
print("changed=%s" % r.changed)
