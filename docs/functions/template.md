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
| `template` | string | `template` or `source` | Path to the template file. A relative path is next to the `.star` file the call is written in. |
| `source` | string | `template` or `source` | The template text itself, instead of a file. A triple-quoted string keeps it readable; see [Render an inline template](#render-an-inline-template). |
| `data` | dict | required | Values the template can use. Keys must be strings; values can be strings, numbers, bools, `None`, lists, tuples, nested dicts and structs. A function's result stands for its value. |
| `destination` | string | required unless `what_if` | The file to write, as an absolute path. Its parent directories are created. |

Give exactly one of `template` and `source`. The template file ships with
your code, so a relative `template` path is found next to the `.star` file
the call is written in, not the directory starcm runs in. `destination` is
an output, so it must be a full path, such as
`"/Library/LaunchDaemons/com.example.agent.plist"`. See
[paths](../common-functionality.md#paths).

`template` also takes the [common arguments](../common-functionality.md#common-arguments): `label`
(required), `only_if`, `not_if`, `timeout` and
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
[examples/template/](../../examples/template).

```scrut
$ cp -R "$TESTDIR/../../examples/template/." "$TMPDIR" && cd "$TMPDIR" && managed="$(pwd -W 2>/dev/null || pwd)"
```

### Render a launchd job

A template for a launchd job, filled in from `data`:

```jinja
{# examples/template/site/agent.plist.tpl #}
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

```python
# examples/template/site/launchd.star
load("starcm", "template", "vars")

# A launchd job that runs the agent's check-in every 15 minutes.
r = template(
    label       = "agent launchd job",
    template    = "agent.plist.tpl",
    destination = vars["managed_dir"] + "/com.example.agent.plist",
    data        = {
        "label":            "com.example.agent",
        "program":          ["/usr/local/bin/agent", "check-in"],
        "interval_seconds": 900,
        "run_at_load":      True,
    },
)
print("changed=%s" % r.changed)
```

`agent.plist.tpl` is found next to `launchd.star`, wherever starcm runs from.
On a Mac, the destination would be
`/Library/LaunchDaemons/com.example.agent.plist`; here it goes in the folder
the run passes as `--var managed_dir`.

```scrut
$ starcm --var managed_dir="$managed" site/launchd.star && cat com.example.agent.plist
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
$ (cd site && starcm --var managed_dir="$managed" launchd.star)
changed=False
```

### Render an inline template

A short template can be written straight into the `.star` file with
`source`, as a triple-quoted string. The backslash after the opening `"""`
skips the first newline, so the template starts at the left margin:

```python
# examples/template/site/inline.star
load("starcm", "template", "vars")

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
    destination = vars["managed_dir"] + "/agent.conf",
)
print("changed=%s" % r.changed)
```

```scrut
$ starcm --var managed_dir="$managed" site/inline.star && cat agent.conf
changed=True
server = https://config.example.com
channel = beta
feature.updates = on
feature.telemetry = on
```

### Preview with `what_if`

Loop over lists and dicts to build a config file. Dict loops run in
alphabetical key order; use a list when order matters.

```jinja
{# examples/template/site/ini.tpl #}
{% for section in sections -%}
[{{ section.name }}]
{% for key, value in section.settings -%}
{{ key }} = {{ value }}
{% endfor %}
{% endfor -%}
```

```python
# examples/template/site/preview.star
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

```jinja
{# examples/template/site/upper.tpl #}
{{ name | upper }} has {{ items | length }} items; first is {{ items | first }}.
{{ missing | default("no value") }}
```

```python
# examples/template/site/filters.star
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

### Fill a template from nested data

`data` can hold nested dicts, lists, tuples and structs, such as
[date](date.md)'s value:

```jinja
{# examples/template/nested/config.tpl #}
[server]
host = {{ server.host }}
port = {{ server.port }}
{% for user in users %}
[user.{{ user.name }}]
groups = {{ user.groups | join(", ") }}
{%- if user.shell %}
shell = {{ user.shell }}
{%- endif %}
{% endfor %}
[build]
date = {{ now.formatted }}
```

```python
# examples/template/nested/nested.star
load("starcm", "date", "template", "write")

# data can hold nested dicts, lists, tuples, None and structs (a result
# from another starcm function stands for its value). Templates reach into them with
# dots or brackets: server.host, users[0]["name"], now.formatted.
render = template(
    label = "render config.tpl",
    template = "config.tpl",
    data = {
        "server": {"host": "localhost", "port": 8080},
        "users": [
            {"name": "ada", "groups": ("admin", "dev"), "shell": "/bin/zsh"},
            {"name": "bob", "groups": ["dev"], "shell": None},
        ],
        "now": date(format = "date", timezone = "UTC"),
    },
    what_if = True,
)
write(render.message, label = "print config", end = "")
```

```scrut
$ (cd nested && starcm nested.star)
[server]
host = localhost
port = 8080

[user.ada]
groups = admin, dev
shell = /bin/zsh

[user.bob]
groups = dev

[build]
date = \d{4}-\d{2}-\d{2} (re)
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/template/edge/](../../examples/template/edge).

```scrut
$ cd "$TMPDIR/edge" && managed="$(pwd -W 2>/dev/null || pwd)"
```

### destination is required unless what_if

```python
# examples/template/edge/missing_destination.star
load("starcm", "template")

template(label = "nowhere", template = "hello.tpl", data = {"name": "world", "age": 42})
```

```scrut
$ starcm missing_destination.star 2>&1
Traceback (most recent call last):
  missing_destination.star:3:9: in <toplevel>
Error in template: destination is required in template() module if what_if is false
[1]
```

### Nested data

`data` can hold nested dicts, lists, tuples, `None` and structs; a result
from another starcm function stands for its value. Dict loops (`for k, v in d`) run in
alphabetical key order; use a list when order matters.

```python
# examples/template/edge/nested.star
load("starcm", "template", "write")
load("stdlib", "struct")

r = template(
    label    = "nested",
    template = "nested.tpl",
    what_if  = True,
    data     = {
        "user": {
            "name": "ada",
            "address": {"city": "london"},
            "pets": [{"kind": "cat", "name": "tom"}, {"kind": "dog", "name": "rex"}],
        },
        "env": {"ZED": "1", "ALPHA": "2"},
        "point": (3, 4),
        "nothing": None,
        "greeting": struct(value = "hello", success = True),
    },
)
write(r.message, label = "show render", end = "")
```

```scrut
$ starcm nested.star
dots: ada lives in london
brackets: ada rex
list: cat=tom dog=rex 
dict: ALPHA=2 ZED=1 
tuple: 3,4
none: [] unset
struct: hello success=True
```

### Dict keys must be strings

```python
# examples/template/edge/non_string_key.star
load("starcm", "template")

template(label = "bad keys", template = "hello.tpl", what_if = True, data = {"name": "x", "age": {1: "one"}})
```

```scrut
$ starcm non_string_key.star 2>&1
Traceback (most recent call last):
  non_string_key.star:3:9: in <toplevel>
Error in template: template() data: "age": dict keys must be strings, got int key 1
[1]
```

### Inline templates

`source` is the template text itself, instead of a template file.

```python
# examples/template/edge/source.star
load("starcm", "template", "write")

r = template(label = "inline", source = "Hello {{ name | capitalize }}.\n", data = {"name": "world"}, what_if = True)
write(r.message, label = "show", end = "")
```

```scrut
$ starcm source.star
Hello World.
```

Exactly one of `template` and `source` must be given.

```python
# examples/template/edge/both.star
load("starcm", "template")

template(label = "both", template = "hello.tpl", source = "Hello {{ name }}", data = {"name": "x"}, what_if = True)
```

```scrut
$ starcm both.star 2>&1
Traceback (most recent call last):
  both.star:3:9: in <toplevel>
Error in template: template() takes template or source, not both
[1]
```

```python
# examples/template/edge/neither.star
load("starcm", "template")

template(label = "neither", data = {"name": "x"}, what_if = True)
```

```scrut
$ starcm neither.star 2>&1
Traceback (most recent call last):
  neither.star:3:9: in <toplevel>
Error in template: template() needs template (a template file) or source (the template text)
[1]
```

## See also

- [file](file.md) for content you build in Starlark.
