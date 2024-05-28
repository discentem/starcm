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
    name               = "ping google a few times",
    cmd                = "ping", 
    args               = ["-n", "google.com"],
    timeout            = "6s",
    live_output        = True
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
result(changed = False, diff = "", error = "timeout 6s exceeded", name = "ping google a few times", output = "", success = False)
```

Now we get a `result` struct! `result` is a [starlarkstruct.Struct](go.starlark.net/starlarkstruct) which we can interact with inside the `.star` file.

<details>
    <summary><h3 style="display:inline-block">Shellout to <code>curl</code></h3></summary>
    <body>
    
</body>
</details>






# Starcm functions

## exec

Example:
```python
exec(
    cmd  = "echo", 
    args = ["hello"],
)
```