# extract

[← All functions](../README.md)

Use `extract` to unpack a `.zip`, `.tar.gz` or `.tgz` archive into a
directory, for example the archive [download](download.md) fetched. The format
comes from the archive's file name. Files already in `dest` with the right
content and permissions are left alone, so `changed` tells you whether
anything happened.

`extract` refuses an archive that could write outside `dest`, before it writes
anything. See [What extract refuses](#what-extract-refuses).

## Syntax

```python
load("starcm", "extract")

extract(
    label   = "unpack terraform",        # required
    archive = terraform.value,           # required; positional or keyword
    dest    = "/usr/local/lib/terraform", # required
    members = ["terraform"],
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `archive` | string | required | The archive to unpack. An absolute path; a leading `~` expands to the home directory. Must end in `.zip`, `.tar.gz` or `.tgz` (any case). It may be the path `download` returned, which `extract` checks against the [local store](download.md#the-local-store) first. |
| `dest` | string | required | The directory to unpack into, created if it's missing. An absolute path; a leading `~` expands to the home directory. |
| `members` | list of strings | all files | Extract only these files, named exactly as the archive spells them (`"tool-1.0/tool"`). A name the archive doesn't hold fails the run. Directories are created as the files need them. |

`extract` also takes the [common arguments](../common-functionality.md#common-arguments): `label`
(required), `only_if`, `not_if`, `timeout` and
`what_if`. `what_if` has no effect.

Files get the permissions the archive records, without setuid, setgid and
sticky bits; an entry that records none gets `0o644`. Windows only keeps a
read-only bit, so there permissions are not compared.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when a file or directory was created or a file rewritten |
| `message` | `extracted N of M files to "<abs dest>"` (N selected of the archive's M files), or `"<abs dest>" already up to date` |
| `value` | A list of the absolute paths of the selected files |

## Examples

Each example runs in a scratch copy of
[examples/extract/](../../examples/extract). The archives in
[examples/extract/files/](../../examples/extract/files) each hold
`tool-1.0/tool`, `tool-1.0/README.txt` and `tool-1.0/docs/LICENSE.txt`.
They were made with Python's `zipfile` and `tarfile` modules, with fixed
timestamps.

```scrut
$ cp -R "$TESTDIR/../../examples/extract/." "$TMPDIR" && cd "$TMPDIR" && managed="$(pwd -W 2>/dev/null || pwd)"
```

### Unpack an archive

```python
# examples/extract/unpack.star
load("starcm", "extract", "here", "vars")

r = extract(
    label   = "unpack tool",
    archive = here("files/tool-1.0.zip"),
    dest    = vars["managed_dir"] + "/opt/tool",
)
print("changed=%s files=%d" % (r.changed, len(r.value)))
```

```scrut
$ starcm --var managed_dir="$managed" unpack.star && cat opt/tool/tool-1.0/README.txt
changed=True files=3
tool 1.0
```

The second run finds every file in place and writes nothing:

```scrut
$ starcm --var managed_dir="$managed" unpack.star
changed=False files=3
```

A file that was changed since is put back:

```scrut
$ echo tampered > opt/tool/tool-1.0/README.txt && starcm --var managed_dir="$managed" unpack.star && cat opt/tool/tool-1.0/README.txt
changed=True files=3
tool 1.0
```

### Unpack a .tar.gz

```python
# examples/extract/unpack_tgz.star
load("starcm", "extract", "here", "vars")

# The format comes from the extension: .zip, .tar.gz or .tgz.
r = extract(here("files/tool-1.0.tar.gz"), dest = vars["managed_dir"] + "/opt/tool-tar", label = "unpack tool")
print("changed=%s files=%d" % (r.changed, len(r.value)))
```

```scrut
$ starcm --var managed_dir="$managed" unpack_tgz.star && cat opt/tool-tar/tool-1.0/docs/LICENSE.txt
changed=True files=3
MIT
```

### Extract only some files

`members` names the files to extract. Only the directories they need are
created, and `value` lists just those files.

```python
# examples/extract/members.star
load("starcm", "extract", "here", "vars")

r = extract(
    label   = "unpack just the tool",
    archive = here("files/tool-1.0.zip"),
    dest    = vars["managed_dir"] + "/bin-only",
    members = ["tool-1.0/tool"],
)
print("changed=%s" % r.changed)
for path in r.value:
    print(path.replace("\\", "/").split("/")[-2:])  # last two elements, for this example
```

```scrut
$ starcm --var managed_dir="$managed" members.star && find bin-only -type f | sed 's|\\|/|g'
changed=True
["tool-1.0", "tool"]
bin-only/tool-1.0/tool
```

### Unpack a download

The examples use a store in the scratch directory; on Windows it goes under
`%LocalAppData%`, as the [exec](exec.md#run-a-file-from-the-local-store) page
explains.

```scrut
$ store="$TMPDIR/store"; case "$(uname -s)" in MINGW*|MSYS*|CYGWIN*) store_root=$(mktemp -d "$(cygpath -u "$LOCALAPPDATA")/starcm-docs.XXXXXX") && store="$store_root/store" ;; esac
```

```python
# examples/extract/from_store.star
load("starcm", "download", "extract", "vars")

tool = download(
    label  = "fetch tool",
    url    = "http://localhost:8000/examples/extract/files/tool-1.0.zip",
    sha256 = "da3bf2e6a81a33c97858840bdfc119be2c127638e62595e18130af760d51a7f9",
)

# tool.value is a path in the store, so extract() re-hashes it before reading.
r = extract(label = "unpack tool", archive = tool.value, dest = vars["managed_dir"] + "/from-store")
print("changed=%s files=%d" % (r.changed, len(r.value)))
```

```scrut
$ starcm --var managed_dir="$managed" --store-dir "$store" from_store.star
changed=True files=3
```

If the stored archive changes after `download` checked it, `extract` refuses
to read it, as [exec](exec.md#run-a-file-from-the-local-store) does:

```python
# examples/extract/from_store_changed.star
load("starcm", "download", "exec", "extract", "vars")

tool = download(
    label  = "fetch tool",
    url    = "http://localhost:8000/examples/extract/files/tool-1.0.zip",
    sha256 = "da3bf2e6a81a33c97858840bdfc119be2c127638e62595e18130af760d51a7f9",
)

# Something changes the stored archive after download() checked it...
exec(label = "tamper", cmd = "sh", args = ["-c", "chmod u+w \"$1\" && echo malware >> \"$1\"", "sh", tool.value])

# ...so extract() refuses to read it.
extract(label = "unpack tool", archive = tool.value, dest = vars["managed_dir"] + "/tampered-out")
```

```scrut
$ starcm --var managed_dir="$managed" --store-dir "$store" from_store_changed.star 2>&1; test ! -e tampered-out && echo "nothing extracted"
Traceback (most recent call last):
  from_store_changed.star:13:8: in <toplevel>
Error in extract: extract(label="unpack tool"): *tool-1.0.zip in the store has sha256 *; it was changed after it was stored (glob)
nothing extracted
```

```scrut
$ rm -rf "$store" ${store_root:+"$store_root"}
```

## What extract refuses

Before it writes anything, `extract` checks every entry in the archive, not
just the ones in `members`. It refuses the whole archive, naming the entry,
if any entry

- has an absolute name: `/x`, `C:\x` or `\\host\x`,
- has a `..` element in its name, whether separated by `/` or `\`, or
- is a symbolic link, hard link, device or other special file.

Files are written through a handle on `dest` that can't leave it, so a
symlink that is already inside `dest` (even one pointing at `/etc`) can't make
`extract` write elsewhere either: the run fails. A symlink where a file
belongs is replaced by the file.

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/extract/edge/](../../examples/extract/edge).

```scrut
$ cd "$TMPDIR/edge" && managed="$(pwd -W 2>/dev/null || pwd)"
```

### An entry that leaves the destination

`evil.zip` holds `ok.txt` and `../evil.txt`. Nothing is written, not even
`ok.txt`, and nothing lands next to `out`.

```python
# examples/extract/edge/traversal.star
load("starcm", "extract", "here", "vars")

# evil.zip holds ok.txt and ../evil.txt.
extract(label = "unpack evil", archive = here("evil.zip"), dest = vars["managed_dir"] + "/out")
```

```scrut
$ starcm --var managed_dir="$managed" traversal.star 2>&1; test ! -e out && test ! -e evil.txt && echo "nothing written"
Traceback (most recent call last):
  traversal.star:4:8: in <toplevel>
Error in extract: extract(label="unpack evil"): refusing entry "../evil.txt": path contains ".."
nothing written
```

### A member the archive doesn't hold

```python
# examples/extract/edge/missing_member.star
load("starcm", "extract", "here", "vars")

extract(
    label   = "unpack missing",
    archive = here("../files/tool-1.0.zip"),
    dest    = vars["managed_dir"] + "/out-missing",
    members = ["tool-1.0/tool", "tool-1.0/nope"],
)
```

```scrut
$ starcm --var managed_dir="$managed" missing_member.star 2>&1; test ! -e out-missing && echo "nothing written"
Traceback (most recent call last):
  missing_member.star:3:8: in <toplevel>
Error in extract: extract(label="unpack missing"): member "tool-1.0/nope" not found in *tool-1.0.zip (glob)
nothing written
```

### An unknown format

The format comes from the file name, not the content.

```python
# examples/extract/edge/unknown_format.star
load("starcm", "extract", "here", "vars")

extract(label = "unpack rar", archive = here("tool.rar"), dest = vars["managed_dir"] + "/out-rar")
```

```scrut
$ starcm --var managed_dir="$managed" unknown_format.star 2>&1
Traceback (most recent call last):
  unknown_format.star:3:8: in <toplevel>
Error in extract: extract(label="unpack rar"): *tool.rar: unknown archive format; extract() handles .zip, .tar.gz and .tgz (glob)
[1]
```

### dest is required

```python
# examples/extract/edge/dest_required.star
load("starcm", "extract", "here")

extract(label = "no dest", archive = here("../files/tool-1.0.zip"))
```

```scrut
$ starcm dest_required.star 2>&1
Traceback (most recent call last):
  dest_required.star:3:8: in <toplevel>
Error in extract: no dest: missing argument for dest
[1]
```

## See also

- [download](download.md) to fetch the archive.
- [file](file.md) to copy a file out of the unpacked archive, optionally
  checking its macOS signature.
- [exec](exec.md) to run something from the unpacked archive.
