# download()

Downloads from the `python3 -m http.server 8000` that the scrut CI job runs
from the repo root, so these tests don't need the internet.

```scrut
$ cp -R "$TESTDIR/fixtures/download/." "$TMPDIR" && cd "$TMPDIR"
```

The store refuses directories that an untrusted account could replace, all
the way up to the root. On the Windows CI runners `D:\` (where `$TMPDIR` is)
is owned by NETWORK SERVICE, so there the store goes under `%LocalAppData%`.

```scrut
$ store="$TMPDIR/store"; case "$(uname -s)" in MINGW*|MSYS*|CYGWIN*) store_root=$(mktemp -d "$(cygpath -u "$LOCALAPPDATA")/starcm-scrut.XXXXXX") && store="$store_root/store" ;; esac
```

## Downloads once, then verifies the existing file

```scrut
$ starcm download.star
first download: success=True changed=True message=downloaded file to downloaded.txt
already present: success=True changed=False message="downloaded.txt" already present and sha256 verified
```

```scrut
$ cat downloaded.txt
starcm download fixture
```

The download is written to a temporary file and renamed into place once its
sha256 matches, so nothing is left behind next to it.

```scrut
$ ls -a | grep -c '\.tmp-'
0
[1]
```

## Creates missing parent directories

```scrut
$ starcm subdir.star && cat nested/dir/payload.txt
success=True changed=True
starcm download fixture
```

## Wrong sha256 fails and removes the download

```scrut
$ rm downloaded.txt && starcm bad_hash.star 2>&1
Traceback (most recent call last):
  bad_hash.star:3:9: in <toplevel>
Error in download: expected sha256 hash 0000000000000000000000000000000000000000000000000000000000000000, got 77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314
[1]
```

```scrut
$ test ! -e downloaded.txt && echo removed
removed
```

## HTTP errors fail the run

```scrut
$ starcm not_found.star 2>&1
Traceback (most recent call last):
  not_found.star:3:9: in <toplevel>
Error in download: failed to download file "http://localhost:8000/tests/fixtures/download/does-not-exist.txt": 404 File not found
[1]
```

## Without save_to, downloads go into the store

Files are stored as `<store>/sha256/<hash>/<name>`, named after the URL unless
`file_name` is given, and `value` is the stored path. A stored file is
re-hashed whenever `download()` finds it already in the store, and whenever
`exec()` is given its path.

```scrut
$ starcm --store-dir "$store" store.star
into the store: changed=True value=*/store/sha256/77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314/payload.txt (glob)
already stored: changed=False value=*/store/sha256/77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314/payload.txt (glob)
named: changed=True value=*/store/sha256/77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314/renamed.txt (glob)
```

```scrut
$ cat "$store"/sha256/77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314/payload.txt
starcm download fixture
```

## A changed file in the store is refused

```scrut
$ f="$store"/sha256/77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314/payload.txt && chmod u+w "$f" && echo edited >> "$f" && starcm --store-dir "$store" store.star 2>&1
Traceback (most recent call last):
  store.star:11:14: in <toplevel>
Error in download: *payload.txt in the store has sha256 *; it was changed after it was stored (glob)
[1]
```

## exec() re-checks stored files before running

Any `cmd` or `args` entry that names a file in the store is re-hashed right
before the command starts, so a file changed after `download()` is refused.

```scrut
$ rm -rf "$store" && starcm --store-dir "$store" store_exec.star 2>&1
read: "starcm download fixture\n"
Traceback (most recent call last):
  store_exec.star:17:4: in <toplevel>
  store_exec.star:10:16: in cat
Error in exec: exec(label="read again"): *payload.txt in the store has sha256 *; it was changed after it was stored (glob)
[1]
```

## --store-only refuses save_to

```scrut
$ starcm --store-only store_only.star 2>&1
Traceback (most recent call last):
  store_only.star:3:9: in <toplevel>
Error in download: download(label="to a path"): save_to is disabled by --store-only; omit it to download into the store
[1]
```

```scrut
$ rm -rf "$store" && starcm --store-only --store-dir "$store" store.star | head -1
into the store: changed=True value=*/store/sha256/77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314/payload.txt (glob)
```

```scrut
$ rm -rf "$store" ${store_root:+"$store_root"}
```
