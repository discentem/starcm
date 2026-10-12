# download()

Downloads from the `python3 -m http.server 8000` that the scrut CI job runs
from the repo root, so these tests don't need the internet.

```scrut
$ cp -R "$TESTDIR/fixtures/download/." "$TMPDIR" && cd "$TMPDIR"
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
