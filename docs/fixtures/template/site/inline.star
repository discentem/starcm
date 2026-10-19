load("starcm", "template")

# A short template can live in the .star file instead of its own file.
r = template(
    label       = "agent settings",
    source      = """\
server = {{ server }}
channel = {{ channel }}
{% for name in features -%}
feature.{{ name }} = on
{% endfor -%}
""",
    data        = {
        "server":   "https://config.example.com",
        "channel":  "beta",
        "features": ["updates", "telemetry"],
    },
    destination = "agent.conf",
)
print("changed=%s" % r.changed)
