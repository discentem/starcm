# file

[← All functions](../README.md)

Use `file` to make sure a file exists with exactly the given content, or as
a copy of another file, or that it doesn't exist at all. The file is only
written when its content differs, so `changed` tells you whether anything
happened.

With `source` and `signer`, `file` installs a macOS binary only if it's
signed by the developer you name, checking the very copy it installs. See
[Copy a signed binary into place](#copy-a-signed-binary-into-place).

## Syntax

```python
load("starcm", "file")

file(
    label       = "agent config",   # required
    path        = "/etc/agent.conf", # required
    content     = "",
    action      = "create",          # or "delete"
    mode        = 0o644,
    create_dirs = False,
)

file(
    label             = "install terraform",  # required
    path              = "/usr/local/bin/terraform",  # required
    source            = "terraform",         # copy this file instead of writing content
    sha256            = "6cb9932b...",
    signer            = "Developer ID Application: Hashicorp, Inc. (D38WU7D763)",
    require_notarized = True,
    mode              = 0o755,
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `path` | string | required | The file to manage, positionally or as `path =`. An absolute path; a leading `~` expands to the home directory. |
| `content` | string | `""` | The file's full content (`action = "create"` only). For several lines, use a triple-quoted string; see [Multi-line content](#multi-line-content). |
| `source` | string | none | Make `path` a copy of this file instead of writing `content`. Give one or the other. An absolute path, like `path`. |
| `sha256` | string | none | With `source`: the source's sha256. A source with any other content is refused. |
| `signer` | string | none | With `source`, macOS only: the signing certificate's name, as [check_mac_signature](check_mac_signature.md#finding-the-signer) describes. The copy is installed only if it passes the same checks. |
| `require_notarized` | bool | `True` | With `signer`: refuse a source Apple hasn't notarized. |
| `action` | string | `"create"` | `"create"` makes the file match `content` or `source`; `"delete"` removes it. |
| `mode` | int | `0o644` | With `content`: permissions for a newly created file; an existing file's permissions are left alone. With `source`: the copy's permissions, set every time; they default to the source's. |
| `create_dirs` | bool | `False` | Create missing parent directories. Without it, a missing parent directory fails the run. |

`file` also takes the [common arguments](../common-functionality.md#common-arguments): `label`
(required), `only_if`, `not_if`, `timeout` and
`what_if`. `what_if` has no effect.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when the file was written or deleted (with `source`: copied, or its mode changed) |
| `message` | One of `created file "<abs path>"`, `file "<abs path>" already exists with correct content`, `copied "<abs source>" to "<abs path>"`, `file "<abs path>" already matches "<abs source>"`, `deleted file "<abs path>"` or `file "<abs path>" does not exist` |
| `value` | With `signer`, the same `mac_signature` struct [check_mac_signature](check_mac_signature.md#returns) returns, its `sha256` being the copy's. Otherwise `None`. |

## Examples

Each example runs in a scratch copy of
[examples/file/](../../examples/file).

```scrut
$ cp -R "$TESTDIR/../../examples/file/." "$TMPDIR" && cd "$TMPDIR" && managed="$(pwd -W 2>/dev/null || pwd)"
```

### Create a file

```python
# examples/file/create.star
load("starcm", "file", "vars")

r = file(
    label   = "agent settings",
    path    = vars["managed_dir"] + "/agent.conf",
    content = "server = https://config.example.com\n",
)
print(r)
```

```scrut
$ starcm --var managed_dir="$managed" create.star && cat agent.conf
starcm_result(changed = True, error = "<nil>", label = "agent settings", message = "created file \"*agent.conf\"", success = True, value = None) (glob)
server = https://config.example.com
```

The second run finds the right content and doesn't touch the file:

```scrut
$ starcm --var managed_dir="$managed" create.star
starcm_result(changed = False, error = "<nil>", label = "agent settings", message = "file \"*agent.conf\" already exists with correct content", success = True, value = None) (glob)
```

### Replace the content

`content` is the whole file, not a line to add.

```python
# examples/file/update.star
load("starcm", "file", "vars")

r = file(
    label   = "agent settings",
    path    = vars["managed_dir"] + "/agent.conf",
    content = """\
server = https://config.example.com
channel = beta
""",
)
print("changed=%s" % r.changed)
```

```scrut
$ starcm --var managed_dir="$managed" update.star && cat agent.conf
changed=True
server = https://config.example.com
channel = beta
```

### Multi-line content

Write multi-line content as a triple-quoted string, as in the example above.
A backslash right after the opening `"""` skips the first newline, so the
content starts on the next line at the left margin, exactly as it will be in
the file:

```python
content = """\
server = https://config.example.com
channel = beta
""",
```

Inside a function, where the lines would be indented with the code, use
[dedent](dedent.md) to take the indentation off.

### Create parent directories and set the mode

```python
# examples/file/create_dirs.star
load("starcm", "file", "vars")

r = file(
    label       = "agent config",
    path        = vars["managed_dir"] + "/etc/agent/agent.conf",
    content     = "server = https://config.example.com\n",
    mode        = 0o600,
    create_dirs = True,
)
print("success=%s changed=%s" % (r.success, r.changed))
```

```scrut
$ starcm --var managed_dir="$managed" create_dirs.star && cat etc/agent/agent.conf
success=True changed=True
server = https://config.example.com
```

Without `create_dirs`, a missing parent directory fails the run:

```python
# examples/file/missing_dir.star
load("starcm", "file", "vars")

file(label = "no parent", path = vars["managed_dir"] + "/nowhere/file.txt", content = "x")
```

```scrut
$ starcm --var managed_dir="$managed" missing_dir.star 2>&1
Traceback (most recent call last):
  missing_dir.star:3:5: in <toplevel>
Error in file: directory ".*[/\\]nowhere" does not exist and create_dirs is false (regex)
[1]
```

### Write to the home directory

Here `HOME` points at the scratch directory so the example doesn't touch your
real home directory.

```python
# examples/file/home.star
load("starcm", "file")

r = file(label = "dotfile", path = "~/.agentrc", content = "debug = false\n")
print("changed=%s" % r.changed)
```

```scrut
$ HOME="$TMPDIR" starcm home.star && cat "$TMPDIR/.agentrc"
changed=True
debug = false
```

### Delete a file

Deleting a file that's already gone succeeds with `changed = False`.

```python
# examples/file/delete.star
load("starcm", "file", "vars")

for i in range(2):
    r = file(label = "remove agent settings", path = vars["managed_dir"] + "/agent.conf", action = "delete")
    print("pass %d: changed=%s message=%s" % (i + 1, r.changed, r.message))
```

```scrut
$ starcm --var managed_dir="$managed" delete.star
pass 1: changed=True message=deleted file "*agent.conf" (glob)
pass 2: changed=False message=file "*agent.conf" does not exist (glob)
```

### Copy a file

With `source`, `file` makes `path` a copy of another file. Here the source is
a file [download](download.md) put in the local store. `file` hashes a store
file again before copying it, as [exec](exec.md#files-from-the-local-store)
does, and `sha256` pins the content too. The examples use a store in the
scratch directory; on Windows it goes under `%LocalAppData%`, as the
[exec](exec.md#run-a-file-from-the-local-store) page explains.

```scrut
$ store="$TMPDIR/store"; case "$(uname -s)" in MINGW*|MSYS*|CYGWIN*) store_root=$(mktemp -d "$(cygpath -u "$LOCALAPPDATA")/starcm-docs.XXXXXX") && store="$store_root/store" ;; esac
```

```python
# examples/file/copy.star
load("starcm", "download", "file", "vars")

agent = download(
    label  = "fetch agent",
    url    = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
    sha256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)

# agent.value is a path in the store, so file() re-hashes it before copying.
r = file(
    label  = "install agent",
    path   = vars["managed_dir"] + "/agent.txt",
    source = agent.value,
    sha256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)
print("changed=%s" % r.changed)
```

```scrut
$ starcm --var managed_dir="$managed" --store-dir "$store" copy.star && cat agent.txt
changed=True
agent v1.2.0
```

The second run finds the same content and copies nothing:

```scrut
$ starcm --var managed_dir="$managed" --store-dir "$store" copy.star
changed=False
```

If the stored file changes after `download` checked it, `file` refuses to
copy it:

```python
# examples/file/copy_changed.star
load("starcm", "download", "exec", "file", "vars")

agent = download(
    label  = "fetch agent",
    url    = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
    sha256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)

# Something changes the stored file after download() checked it...
exec(label = "tamper", cmd = "sh", args = ["-c", "chmod u+w \"$1\" && echo malware >> \"$1\"", "sh", agent.value])

# ...so file() refuses to copy it.
file(label = "install agent", path = vars["managed_dir"] + "/agent.txt", source = agent.value)
```

```scrut
$ starcm --var managed_dir="$managed" --store-dir "$store" copy_changed.star 2>&1
Traceback (most recent call last):
  copy_changed.star:13:5: in <toplevel>
Error in file: file(label="install agent"): *agent-1.2.0.txt in the store has sha256 *; it was changed after it was stored (glob)
[1]
```

```scrut
$ rm -rf "$store" ${store_root:+"$store_root"}
```

### Copy a signed binary into place

With `signer`, `file` copies the source into a private directory next to
`path`, checks that copy's signature with the same checks as
[check_mac_signature](check_mac_signature.md), and only then moves it into
place. Nothing can swap the file between the check and the install, and a
binary that fails a check is never installed. The signature is checked on
every run, even when `path` already matches.

On macOS these examples install `/bin/ls`, which Apple signs, and refuse a
copy of it with bytes added after it was signed. On Linux and Windows they
fail because `signer` only works on macOS. Installing a Developer ID binary
such as Terraform is shown in
[tests/integration/macos/terraform.md](../../tests/integration/macos/terraform.md).

```scrut
$ if command -v codesign >/dev/null; then cp /bin/ls ls-tampered && printf junk >> ls-tampered; else echo not code > ls-tampered; fi
```

```python
# examples/file/install_signed.star
load("starcm", "file", "vars")

# The copy is checked in a private directory next to path, then moved into
# place, so what's installed is exactly what was checked.
r = file(
    label             = "install tool",
    path              = vars["managed_dir"] + "/bin/tool",
    source            = vars["source"],
    signer            = "macOS Software Signing",
    require_notarized = False,  # Apple doesn't notarize its own code
    mode              = 0o755,
    create_dirs       = True,
)
print("changed=%s signer=%s" % (r.changed, r.value.signer))
```

```scrut
$ starcm --var managed_dir="$managed" --var source=/bin/ls install_signed.star 2>&1 | tail -1
(changed=True signer=macOS Software Signing|Error in file: file\(label="install tool"\): signer only works on macOS) (regex)
```

```scrut
$ starcm --var managed_dir="$managed" --var source=/bin/ls install_signed.star 2>&1 | tail -1
(changed=False signer=macOS Software Signing|Error in file: file\(label="install tool"\): signer only works on macOS) (regex)
```

A modified binary is refused, and `bin/tool` is left as it was:

```scrut
$ starcm --var managed_dir="$managed" --var source="$PWD/ls-tampered" install_signed.star 2>&1 | grep '^Error'; if [ -e bin/tool ]; then cmp -s bin/tool /bin/ls && echo "bin/tool unchanged"; else echo "bin/tool unchanged"; fi; ls -a bin 2>/dev/null | grep starcm-file || echo "nothing left behind"
Error in file: file\(label="install tool"\): (.*ls-tampered: codesign --verify: exit status 1|signer only works on macOS) (regex)
bin/tool unchanged
nothing left behind
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/file/edge/](../../examples/file/edge).

```scrut
$ cd "$TMPDIR/edge" && managed="$(pwd -W 2>/dev/null || pwd)"
```

### A pinned sha256 must match

```python
# examples/file/edge/copy_wrong_hash.star
load("starcm", "file", "here", "vars")

file(
    label  = "pinned",
    path   = vars["managed_dir"] + "/copy.txt",
    source = here("source.txt"),
    sha256 = "0000000000000000000000000000000000000000000000000000000000000000",
)
```

```scrut
$ starcm --var managed_dir="$managed" copy_wrong_hash.star 2>&1; test ! -e copy.txt && echo "nothing copied"
Traceback (most recent call last):
  copy_wrong_hash.star:3:5: in <toplevel>
Error in file: file(label="pinned"): *source.txt has sha256 *, not the expected 0000000000000000000000000000000000000000000000000000000000000000 (glob)
nothing copied
```

### content and source don't go together

```python
# examples/file/edge/copy_and_content.star
load("starcm", "file", "here", "vars")

file(label = "both", path = vars["managed_dir"] + "/copy.txt", source = here("source.txt"), content = "x")
```

```scrut
$ starcm --var managed_dir="$managed" copy_and_content.star 2>&1
Traceback (most recent call last):
  copy_and_content.star:3:5: in <toplevel>
Error in file: file(label="both"): give content or source, not both
[1]
```

### signer needs source

```python
# examples/file/edge/signer_without_source.star
load("starcm", "file", "vars")

file(
    label   = "no source",
    path    = vars["managed_dir"] + "/tool",
    content = "x",
    signer  = "Developer ID Application: Example Corp (ABCDE12345)",
)
```

```scrut
$ starcm --var managed_dir="$managed" signer_without_source.star 2>&1
Traceback (most recent call last):
  signer_without_source.star:3:5: in <toplevel>
Error in file: file(label="no source"): signer needs source
[1]
```

## See also

- [template](template.md) to render a file's content from a template.
- [json](json.md) to write JSON content.
- [extract](extract.md) to unpack an archive before copying a file out of it.
- [check_mac_signature](check_mac_signature.md) to report a signature
  without installing anything.
