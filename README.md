# starcm

- A rudimentary config management language that utilizes Starlark, specifically [starlark-go](github.com/google/starlark-go) for configuration instead of json or yaml. 
- Starcm is not full replacement for tools like Chef or Ansible, but starcm can be used to bootstrap those tools and many others!

# Intro to starcm language

## `exec`
<details>
    <summary><h3 style="display:inline-block">Shellout to <code>curl</code></h3></summary>
<body>
    
Let's look at an example starcm configuration file that uses the `exec` function: [ping_apple.star](examples/ping_apple.star)


This configuration will simply shell out to curl and ping [apple.com](apple.com).

```python
% cat examples/ping_apple.star 
load("shellout", "exec")
a = exec(
    name               = "ping google a few times",
    cmd                = "ping", 
    args               = ["-n", "google.com"],
    timeout            = "3s",
    live_output        = True
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
    <summary><h3 style="display:inline-block">Shellout to <code>curl</code> with a timeout</h3></summary>
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

Now we get a `result` struct! Generally all starcm functions return this result struct, which we can interact with inside the `.star` file. But let's put that aside for a moment because we got a failure (e.g. **`success = False`**).

For now, let's see what a successful `exec` looks like with a simple echo command:
</body>
</details>

<body>
<details>
<summary><h3 style="display:inline-block"><code>echo</code></h3></summary>

See [examples/ping_google_with_timeout.star](examples/ping_google_with_timeout.star).

```python
% cat examples/echo.star
load("shellout", "exec")

a = exec(
    name = "echo_hello",
    cmd  = "echo", 
    args = ["hello world!"],
)
print(a)
```

We can execute it with

```shell
go run main.go --root_file examples/echo.star
```

```shell
INFO: 2024/05/29 22:30:19 starting starcm...
INFO: 2024/05/29 22:30:19 [echo_hello]: Starting...
result(changed = True, diff = "", error = None, name = "echo_hello", output = "hello world!\n", success = True)
```

We can even have a success if the exit code is not expected to be zero.

</details>
</body>

<body>
<details>
<summary><h3 style="display:inline-block">expect non-zero exit</h3></summary>

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