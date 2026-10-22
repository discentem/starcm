# dedent

[← All functions](../README.md)

`dedent` lets a multi-line string be indented with the code around it. It
takes the text of a triple-quoted string and removes:

- the newline right after the opening `"""`;
- the indentation all non-blank lines share, so lines indented more than the
  rest keep the extra;
- the indentation before the closing `"""`, keeping the newline before it.

Use it for file content written inside a function, or anywhere a
triple-quoted string would otherwise have to start at the left margin. Like
[json](json.md) and [temp_dir](temp_dir.md), it's a plain helper: it takes no
`label` and returns a string.

At the top level of a file you often don't need it: start the string with
`"""\` (a backslash right after the quotes) to skip the first newline, and
write the lines at the left margin. See [file](file.md#multi-line-content).

## Syntax

```python
load("starcm", "dedent")

dedent("""
    server = https://config.example.com
    channel = stable
    """)    # "server = https://config.example.com\nchannel = stable\n"
```

## Arguments

| Argument | Type | Description |
| --- | --- | --- |
| `text` | string | The text to dedent, positionally. |

## Returns

The text without the leading newline and the shared indentation. Lines that
are blank or only whitespace become empty. Tabs and spaces count as different
characters, so mix them consistently.

## Examples

```scrut
$ cp -R "$TESTDIR/../../examples/dedent/." "$TMPDIR" && cd "$TMPDIR" && managed="$(pwd -W 2>/dev/null || pwd)"
```

### Write a settings file from a function

```python
# examples/dedent/settings.star
load("starcm", "dedent", "file", "vars")

def agent_settings(channel):
    # Indented with the code; dedent() takes the indentation off.
    return dedent("""
        server = https://config.example.com
        channel = %s
        [logging]
            level = info
        """ % channel)

r = file(label = "agent settings", path = vars["managed_dir"] + "/agent.conf", content = agent_settings("beta"))
print("changed=%s" % r.changed)
```

```scrut
$ starcm --var managed_dir="$managed" settings.star && cat agent.conf
changed=True
server = https://config.example.com
channel = beta
[logging]
    level = info
```

## See also

- [file](file.md) and [template](template.md#render-an-inline-template), which take multi-line strings.
