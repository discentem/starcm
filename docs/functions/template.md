# template

[← All functions](../README.md)

Use `template` to render a [Jinja2-style](https://jinja.palletsprojects.com/)
template with data from your configuration and write the result to a file.
The file is only written when the rendered text differs from what's there.
With `what_if = True` it renders without writing, so you can preview the
result.

Templates are rendered by [gonja](https://github.com/noirbizarre/gonja), a Go
implementation of Jinja2: `{{ value }}`, `{% if %}`, `{% for %}`, filters such
as `upper`, `join` and `default`, and `-` to trim whitespace all work. A
template's trailing newline is kept.

## Syntax

```python
load("starcm", "template")

template(
    label       = "agent launchd job",                               # required
    template    = "agent.plist.tpl",                                 # template or source is required
    source      = """...""",                                         # the template text, instead of a file
    data        = {"label": "com.example.agent"},                    # required
    destination = "/Library/LaunchDaemons/com.example.agent.plist",  # required unless what_if = True
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `template` | string | `template` or `source` | Path to the template file. |
| `source` | string | `template` or `source` | The template text itself, instead of a file. A triple-quoted string keeps it readable; see [Render an inline template](#render-an-inline-template). |
| `data` | dict | required | Values the template can use. Keys must be strings; values can be strings, numbers, bools, `None`, lists, tuples, nested dicts and structs (including other functions' results). |
| `destination` | string | required unless `what_if` | The file to write. Its parent directories are created. |

Give exactly one of `template` and `source`. Relative `template` and
`destination` paths are relative to the directory of the `.star` file that
calls `template`, not the directory starcm runs in. Set
[working_directory](../common-functionality.md#working_directory) to use a
different base directory.

`template` also takes the [common arguments](../common-functionality.md#common-arguments): `label`
(required), `only_if`, `not_if`, `timeout`, `working_directory` and
`what_if`. `what_if = True` renders the template but writes nothing.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when `destination` was (or, with `what_if`, would be) written |
| `message` | The rendered text |
| `value` | `None` |

Template syntax errors and missing files stop the run.

## Examples

Each example runs in a scratch copy of
[docs/fixtures/template/](../fixtures/template).

```scrut
$ cp -R "$TESTDIR/../fixtures/template/." "$TMPDIR" && cd "$TMPDIR"
```

### Render a launchd job

A template for a launchd job, filled in from `data`:

<!-- fixture: ../fixtures/template/site/agent.plist.tpl -->
```jinja
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{ label }}</string>
    <key>ProgramArguments</key>
    <array>
{%- for arg in program %}
        <string>{{ arg }}</string>
{%- endfor %}
    </array>
    <key>StartInterval</key>
    <integer>{{ interval_seconds }}</integer>
{%- if run_at_load %}
    <key>RunAtLoad</key>
    <true/>
{%- endif %}
</dict>
</plist>
```

<!-- fixture: ../fixtures/template/site/launchd.star -->
```python
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
```

`agent.plist.tpl` and `com.example.agent.plist` are found next to
`launchd.star`, wherever starcm runs from. On a Mac, the destination would be
`/Library/LaunchDaemons/com.example.agent.plist`; here it stays in the
example's directory.

```scrut
$ starcm site/launchd.star && cat site/com.example.agent.plist
changed=True
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.example.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/agent</string>
        <string>check-in</string>
    </array>
    <key>StartInterval</key>
    <integer>900</integer>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
```

Rendering the same data again leaves the file alone:

```scrut
$ (cd site && starcm launchd.star)
changed=False
```

### Render an inline template

A short template can be written straight into the `.star` file with
`source`, as a triple-quoted string. The backslash after the opening `"""`
skips the first newline, so the template starts at the left margin:

<!-- fixture: ../fixtures/template/site/inline.star -->
```python
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
```

```scrut
$ starcm site/inline.star && cat site/agent.conf
changed=True
server = https://config.example.com
channel = beta
feature.updates = on
feature.telemetry = on
```

### Preview with `what_if`

Loop over lists and dicts to build a config file. Dict loops run in
alphabetical key order; use a list when order matters.

<!-- fixture: ../fixtures/template/site/ini.tpl -->
```jinja
{% for section in sections -%}
[{{ section.name }}]
{% for key, value in section.settings -%}
{{ key }} = {{ value }}
{% endfor %}
{% endfor -%}
```

<!-- fixture: ../fixtures/template/site/preview.star -->
```python
load("starcm", "template", "write")

r = template(
    label    = "preview agent.ini",
    template = "ini.tpl",
    what_if  = True,
    data     = {
        "sections": [
            {"name": "server", "settings": {"host": "config.example.com", "port": 443}},
            {"name": "logging", "settings": {"level": "info"}},
        ],
    },
)
print("changed=%s" % r.changed)
write(r.message, label = "show render", end = "")
```

```scrut
$ starcm site/preview.star
changed=True
[server]
host = config.example.com
port = 443

[logging]
level = info

```

### Filters

<!-- fixture: ../fixtures/template/site/upper.tpl -->
```jinja
{{ name | upper }} has {{ items | length }} items; first is {{ items | first }}.
{{ missing | default("no value") }}
```

<!-- fixture: ../fixtures/template/site/filters.star -->
```python
load("starcm", "template", "write")

r = template(
    label    = "filters",
    template = "upper.tpl",
    what_if  = True,
    data     = {"name": "fleet", "items": ["mac", "win"]},
)
write(r.message, label = "show render", end = "")
```

```scrut
$ starcm site/filters.star
FLEET has 2 items; first is mac.
no value
```

## See also

- [file](file.md) for content you build in Starlark.
- [tests/template.md](../../tests/template.md) for nested data, structs and errors.
