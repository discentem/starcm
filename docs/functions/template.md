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
    label       = "motd",           # required
    template    = "motd.tpl",       # required
    data        = {"name": "ada"},  # required
    destination = "/etc/motd",      # required unless what_if = True
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `template` | string | required | Path to the template file. |
| `data` | dict | required | Values the template can use. Keys must be strings; values can be strings, numbers, bools, `None`, lists, tuples, nested dicts and structs (including other functions' results). |
| `destination` | string | required unless `what_if` | The file to write. Its parent directories are created. |

Relative `template` and `destination` paths are relative to the directory of
the `.star` file that calls `template`, not the directory starcm runs in. Set
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

### Render a file

<!-- fixture: ../fixtures/template/site/motd.tpl -->
```jinja
Welcome to {{ hostname }}.
{% if maintenance -%}
Maintenance window: {{ maintenance }}
{% endif -%}
Admins: {{ admins | join(", ") }}
```

<!-- fixture: ../fixtures/template/site/motd.star -->
```python
load("starcm", "template")

r = template(
    label       = "motd",
    template    = "motd.tpl",
    destination = "motd.txt",
    data        = {
        "hostname":    "build-01",
        "maintenance": "Sundays 02:00 UTC",
        "admins":      ["ada", "grace"],
    },
)
print("changed=%s" % r.changed)
```

`motd.tpl` and `motd.txt` are found next to `motd.star`, wherever starcm runs
from:

```scrut
$ starcm site/motd.star && cat site/motd.txt
changed=True
Welcome to build-01.
Maintenance window: Sundays 02:00 UTC
Admins: ada, grace
```

Rendering the same data again leaves the file alone:

```scrut
$ (cd site && starcm motd.star)
changed=False
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
