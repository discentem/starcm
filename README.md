# starcm

- A rudimentary config management language that utilizes Starlark, specifically [starlark-go](github.com/google/starlark-go) for configuration instead of json or yaml. 
- Starcm is not full replacement for tools like Chef or Ansible, but starcm can be used to bootstrap those tools and many others!

# Intro to starcm language

## Shelling out

Let's look at a simple starcm file that uses the `exec` function: [examples/echo.star](examples/echo.star)

```python
% cat examples/echo.star
load("shellout", "exec")
exec(
    name               = "hello_from_starcm",
    cmd                = "echo", 
    args               = ["hello from echo.star!"],
    live_output        = True
)
```

We can execute it with

```shell
go run main.go --root_file examples/echo.star
```

which outputs

```shell
% go run main.go --root_file examples/echo.star
INFO: 2024/06/01 23:46:17 starting starcm...
INFO: 2024/06/01 23:46:17 [hello_from_starcm]: Starting...
hello from echo.star!
```

<details>
    <summary><h3 style="display:inline-block">long running commands with live output</h3></summary>
<body>

This configuration will simply shell out to ping and ping [apple.com](apple.com). See [ping_apple.star](examples/ping_apple.star)

```python
% cat examples/ping_apple.star 
load("shellout", "exec")
a = exec(
    name               = "ping google a few times",
    cmd                = "ping", 
    args               = ["-n", "google.com"],
    timeout            = "3s",
    live_output        = True # causes stdout to appear live
)
print(a)
```

We can execute it with

```shell
go run main.go --root_file examples/ping_apple.star
```

```
INFO: 2024/05/27 15:06:58 [ping apple.com]: Starting...
64 bytes from 17.253.144.10: icmp_seq=3 ttl=56 time=19.419 ms
64 bytes from 17.253.144.10: icmp_seq=4 ttl=56 time=17.524 ms
...
64 bytes from 17.253.144.10: icmp_seq=8 ttl=56 time=17.621 ms
```

But you might notice a problem: `ping -n apple.com` never exits! We can handle this by setting a timeout:
</body>
</details>

<details>
    <summary><h3 style="display:inline-block">with a timeout</h3></summary>
<body>
    
See [examples/ping_google_with_timeout.star](examples/ping_google_with_timeout.star).

```python
% cat examples/ping_apple.star 
load("shellout", "exec")
a = exec(
    name        = "ping google a few times",
    cmd         = "ping", 
    args        = ["-n", "google.com"],
    timeout     = "6s",
    live_output = True
)
print(a)
```

We can execute it with

```shell
go run main.go --root_file examples/ping_google_with_timeout.star
```

```
...
64 bytes from 142.251.214.142: icmp_seq=0 ttl=116 time=16.926 ms
64 bytes from 142.251.214.142: icmp_seq=1 ttl=116 time=20.704 ms
...
64 bytes from 142.251.214.142: icmp_seq=5 ttl=116 time=20.717 ms
result(changed = False, diff = "", error = "context deadline exceeded", name = "ping google a few times", output = "PING apple.com (17.253.144.10): 56 data bytes\n64 bytes from 17.253.144.10: icmp_seq=0 ttl=56 time=16.329 ms\n64 bytes from 17.253.144.10: icmp_seq=1 ttl=56 time=21.740 ms\n64 bytes from 17.253.144.10: icmp_seq=2 ttl=56 time=22.659 ms\n64 bytes from 17.253.144.10: icmp_seq=3 ttl=56 time=20.311 ms\n64 bytes from 17.253.144.10: icmp_seq=4 ttl=56 time=20.397 ms\n64 bytes from 17.253.144.10: icmp_seq=5 ttl=56 time=20.845 ms\n", success = False)
```

Because the command actually finished and we are printing `a` with `print(a)` we get a `result` struct! Generally all starcm functions return this result struct. We'll explore this in further detail later.

For now, we'll see how we can deal with non-zero exit codes.

</details>
</body>

<body>
<details>
<summary><h3 style="display:inline-block">handling non-zero exit codes</h3></summary>

See [examples/expect_exit_code_non_zero.star](examples/expect_exit_code_non_zero.star)

```python
load("shellout", "exec")
a = exec(
    name               = "explicitly exit 2",
    cmd                = "sh", 
    args               = ["-c", "echo 'we expect to exit 2'; exit 2"],
    # expected_exit_code = 2,
)
print(a)
```

```shell
INFO: 2024/05/29 22:49:07 starting starcm...
INFO: 2024/05/29 22:49:07 [explicitly exit 2]: Starting...
result(changed = True, diff = "", error = "exit status 2", name = "explicitly exit 2", output = "we expect to exit 2\n", success = False)
```

This would fail because the default `expected_error_code` is `0`. But if we set it to `2` then this succeeds!

```python
load("shellout", "exec")
a = exec(
    name               = "explicitly exit 2",
    cmd                = "sh", 
    args               = ["-c", "echo 'we expect to exit 2'; exit 2"],
    expected_exit_code = 2,
)
print(a)
```

```shell
INFO: 2024/05/29 22:51:04 starting starcm...
INFO: 2024/05/29 22:51:04 [explicitly exit 2]: Starting...
result(changed = True, diff = "", error = "exit status 2", name = "explicitly exit 2", output = "we expect to exit 2\n", success = True)
```

</details>
</body>

<body>
<details>
<summary><h3 style="display:inline-block">if statements</h3></summary>

Starlark, and by extension starcm, supports `if` statements. Take [examples/if_statements.star](examples/if_statements.star) for example:

```python
load("shellout", "exec")
a = exec(
    name               = "explicitly exit 2",
    cmd                = "sh", 
    args               = ["-c", "echo 'we expect to exit 2'; exit 2"],
    expected_exit_code = 2,
)
if a.success == True:
    print("party!")
else:
    print("no party :(")
```

Running `go run main.go --root_file examples/if_statements.star` results in

```shell
% go run main.go --root_file examples/if_statements.star 
INFO: 2024/06/01 23:52:56 starting starcm...
INFO: 2024/06/01 23:52:56 [explicitly exit 2]: Starting...
party!
```

We can also implement this same conditional behavior with a starcm construct called `only_if`.

</details>
</body>

<body>
<details>
<summary><h3 style="display:inline-block">only_if</h3></summary>

See [examples/only_if.star](examples/only_if.star)

</details>
</body>

<body>
<details>
<summary><h3 style="display:inline-block">extra</h3></summary>

</details>
</body>









# Starcm functions

## exec

Example:
```python
exec(
    cmd  = "echo", 
    args = ["hello"],
)
```