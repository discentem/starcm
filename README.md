![Static Badge](https://img.shields.io/badge/under%20development%2C%20not%20production%20ready-red?labelColor=yellow)

# starcm
"star-cm"

- A rudimentary configuration management language that utilizes Starlark instead of json or yaml.
- Why Starlark? It provides variables, functions, loops, and lots more "for free" inside of the configuration files!
- Starcm is not intended to be a full replacement for tools like Chef or Ansible, but starcm can be used to bootstrap these tools and many others through features like `exec()` for calling binaries, `template()` for rendering templated files, `load_dynamic()` for loading additional starcm config files dynamically, and much more!

# Goal

Starcm is intended to become a viable alternative for tools like [macadmins/installapplications](https://github.com/macadmins/installapplications), [facebookincubator/go2chef](https://github.com/facebookincubator/go2chef), and [google/glazier](https://github.com/google/glazier).

# Introduction to the starcm language (functions)

## exec

Let's look at a simple starcm file that calls out to `echo`: [examples/echo.star](examples/echo.star)

https://github.com/discentem/starcm/blob/6d679d49b26b63cef277a33c0cd96861e131fb9e/examples/echo/echo.star#L1

When we run this, we see the string we passed to `args` get printed out:

```scrut
$ bazel run :starcm -- --root_file examples/echo/echo.star --timestamps=false
INFO: starting starcm...
INFO: [hello_from_echo_dot_star]: Executing...
hello from echo.star!
```

This is a very trivial example of what Starcm can do. Let's make it a bit more complicated...

Starcm's `exec` can also handle non-zero exit codes.

<body>
<details>
<summary><h3 style="display:inline-block">handling non-zero exit codes</h3></summary>

See [examples/exec/exit_codes/unexpected.star](examples/exec/exit_codes/unexpected.star). 

If `exec` exits with a non-zero exit code there will be a failure returned (`result(..., success=False)`) because the default `expected_error_code` is `0`.

```scrut
$ bazel run :starcm -- --root_file examples/exec/exit_codes/unexpected.star --timestamps=false
INFO: starting starcm...
INFO: [explicitly exit 2]: Executing...
we expect to exit 2
result(changed = True, diff = "", error = "exit status 2", name = "explicitly exit 2", output = "we expect to exit 2\n", success = False)
```

But if we set `expected_exit_code` to `2` then this succeeds!

```scrut
$ bazel run :starcm -- --root_file examples/exec/exit_codes/expected.star --timestamps=false
INFO: starting starcm...
INFO: [explicitly exit 2]: Executing...
we expect to exit 2
result(changed = True, diff = "", error = "exit status 2", name = "explicitly exit 2", output = "we expect to exit 2\n", success = True)
```

Nearly all starcm functions return this `result()` struct which we can combine with conditionals to create powerful and flexible workflows. 

For example, the starcm function `template` also returns a result struct.

</details>
</body>

## template

## Common Functionality

All starcm functions generally return a `result` struct.

### Conditionals

<body>
<details>
<summary><h3 style="display:inline-block">if statements</h3></summary>

Starlark, and by extension starcm, supports `if` statements. Take [examples/if_statements/if_statements.star](examples/if_statements/if_statements.star) for example. If the `exec()` succeeds, we print `party!`. 

```python
load("starcm", "exec")
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

Running `go run main.go --root_file examples/if_statements/if_statements.star` results in

```shell
% go run main.go --root_file examples/if_statements.star 
INFO: 2024/06/01 23:52:56 starting starcm...
INFO: 2024/06/01 23:52:56 [explicitly exit 2]: Starting...
party!
```

We can also implement this same conditional behavior with a starcm-specific construct called `only_if`.

### only_if

</details>
</body>

<body>
<details>
<summary><h3 style="display:inline-block">only_if</h3></summary>

See [examples/only_if/only_if.star](examples/only_if/only_if.star):

```python
load("shellout", "exec")
load("write", "write")

a = exec(
    name               = "explicitly exit 2",
    cmd                = "sh", 
    args               = ["-c", "echo 'we expect to exit 2'; exit 2"],
    expected_exit_code = 2,
    live_output        = True,
)

if not(a.success):
    write(
        name = "print_not_success_#1",
        str = "a.success: %s #1" % (a.success),
    )

write(
    name = "print_not_success_#2",
    str = "a.success: %s #2" % (a.success),
    only_if = a.success == False
)
```

In this example

```python
if not(a.success):
    write(
        name = "print_not_success_#1",
        str = "a.success: %s #1" % (a.success),
    )
```

is essentially equivalent to

```python
write(
    name = "print_not_success_#2",
    str = "a.success: %s #2" % (a.success),
    only_if = a.success == False
)
```

with one key difference: `only_if` produces a log message indicating that `write(name=print_not_success, ...)` was skipped due to the `only_if` condition being false. This is can be useful for debugging.

```bash
% go run main.go -v 2 --root_file examples/only_if.star
INFO: 2024/06/04 23:04:00 starting starcm...
INFO: 2024/06/04 23:04:00 [LoadFromFile]: loading file "examples/only_if.star"
INFO: 2024/06/04 23:04:00 [explicitly exit 2]: Executing...
we expect to exit 2
INFO: 2024/06/04 23:04:00 [explicitly exit 2]: expectedExitCode: 2
INFO: 2024/06/04 23:04:00 [explicitly exit 2]: actualExitCode: 2
INFO: 2024/06/04 23:04:00 [print_not_success_#2]: skipping write(name="print_not_success_#2") because only_if was false
```

> Notice that there is no log message regarding `print_not_success_#1`. Normal `if` statements are not executed at all if the condition is false, whereas `only_if` logs that `print_not_success_#2` was skipped.

</details>
</body>

# Advanced functionality

See the [examples](examples/) folder for more examples of what starcm can do. There's lots it can do such as downloading files (with hash checking), dynamically loading additional `.star` files, rendering templates, and combining all the cabilities via macros, thanks to Starlark.
