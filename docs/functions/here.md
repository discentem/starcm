# here

[← All functions](../README.md)

`here` returns the directory of the `.star` file the `here()` call is written
in, optionally with more path parts joined on. Use it for read-only files that
ship next to your code, such as templates, so the path doesn't depend on the
directory starcm was started in. It isn't for outputs: files a run produces
have no reason to sit next to your code, so give those a full path.

It resolves next to the file where `here()` itself is written, the way
[load()](../loading.md) resolves paths. A helper in another module that calls
`here()` gets its own directory; to point a helper at a file next to *your*
`.star` file, call `here()` yourself and pass the result:

```python
# site/web.star: motd.jinja is next to this file
load("//lib/render.star", "render")
render(here("motd.jinja"), "/etc/motd")
```

Like [home_dir](home_dir.md) and [temp_dir](temp_dir.md), it's a plain helper:
it takes no `label` and returns a string, not a `starcm_result`. It doesn't
check that the file exists. See the
[design note](../design/paths-and-arguments.md) for how starcm treats paths.

## Syntax

```python
load("starcm", "here")

here()                            # e.g. "/repo/site"
here("templates", "motd.jinja")   # e.g. "/repo/site/templates/motd.jinja"
```

## Arguments

| Argument | Type | Description |
| --- | --- | --- |
| `*parts` | strings | Relative path parts to join onto the `.star` file's directory, with the platform's separator (`/`, or `\` on Windows). `..` is allowed. |

## Returns

The absolute path as a string, with no trailing separator.

## Examples

The examples run in a scratch copy of [examples/here/](../../examples/here).

```scrut
$ cp -R "$TESTDIR/../../examples/here/." "$TMPDIR" && cd "$TMPDIR"
```

### Files next to the caller and next to a library

`site/web.star` loads a helper from `lib/render.star` that returns a file
bundled with the library.

```python
# examples/here/site/web.star
load("starcm", "here")
load("../lib/render.star", "bundled_banner")

root = here("..")

def show(path):
    # Print paths relative to the example, with / even on Windows.
    return path[len(root) + 1:].replace("\\", "/")

print(show(here("motd.jinja")))  # next to this file
print(show(bundled_banner()))    # next to lib/render.star, where its here() call is
```

```python
# examples/here/lib/render.star
load("starcm", "here")

def bundled_banner():
    # here() resolves next to this file, wherever it's called from.
    return here("banner.jinja")
```

```scrut
$ cd site && starcm web.star
site/motd.jinja
lib/banner.jinja
```

## Errors and edge cases

These are here mostly as tests. They run in the copy of
[examples/here/edge/](../../examples/here/edge).

```scrut
$ cd "$TMPDIR/edge"
```

### Path parts must be strings

```python
# examples/here/edge/bad_part.star
load("starcm", "here")

here("templates", 1)
```

```scrut
$ starcm bad_part.star 2>&1
Traceback (most recent call last):
  bad_part.star:3:5: in <toplevel>
Error in here: here: argument 2: got int, want string
[1]
```

Parts must also be relative (`here("/etc/motd")` is an error), and `here()`
is an error in modules loaded with [load_remote](load_remote.md): they run
from starcm's store, where nothing ships next to them.

## See also

- [home_dir](home_dir.md) and [temp_dir](temp_dir.md), the other path helpers.
- [Loading additional files](../loading.md) for how `load()` resolves paths.
