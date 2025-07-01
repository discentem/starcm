![Static Badge](https://img.shields.io/badge/under%20development%2C%20not%20production%20ready-red?labelColor=yellow)

# starcm
"star-cm" ⭐

- A rudimentary configuration management language that utilizes Starlark instead of Ruby, json, or yaml.
- Why Starlark? Starlark provides variables, functions, loops, and lots more "for free" inside of the configuration files!

# Goal
Starcm is intended to become a viable alternative for tools like [macadmins/installapplications](https://github.com/macadmins/installapplications), [facebookincubator/go2chef](https://github.com/facebookincubator/go2chef), and [google/glazier](https://github.com/google/glazier).

# Installation

#### Option 1: Download a precompiled release

Download starcm from https://github.com/discentem/starcm/releases and install it somewhere in your path, such as `/usr/local/bin/starcm`.

#### Option 2: Compile 

Install [https://github.com/bazelbuild/bazelisk](Bazelisk) and do `make install`.

# What's possible with Starcm?

## Download & install a pkg

We can use Starcm to download and install packages for macOS. We can even store the configuration file (that tells Starcm what package we want to install) on a web server as well.

See [examples/install_go/bootstrap.star](examples/install_go/bootstrap.star) which shows an example of this.

You can run the example like so:

1. Start a web server that serves this repo.

    ```bash
    $ python3 -m http.server -d .
    ```
1. Run starcm with bootstrap.star, which will download additional `.star` files from the webserver and execute them.

    ```scrut
    $ starcm --root_file examples/install_a_pkg_from_server/bootstrap.star
    result(changed = True, diff = "", error = None, name = "download examplejson-1.0.pkg", output = "downloaded file to examplejson-1.0.pkg", success = True)

    {
        "name": "install examplejson-1.0.pkg",
        "changed": True,
        "success": True
    }
    ```

1. See that `/opt/example.json` now exists.


## shelling out

Let's look at a simple starcm file that calls out to the `echo` binary using a Starcm function called `exec`. This is similar to Chef's `exec` resource.

<!-- Github Markdown engine will render this link as a code snippet. -->

https://github.com/discentem/starcm/blob/2911aea91ad6c978b94b1c237fe4fb38e69b32e2/examples/echo/echo.star#L1-L7

When we run this, we see the string we passed to `args` get printed out:

```scrut
$ bazel run :starcm -- --root_file examples/echo/echo.star --timestamps=false
hello from echo.star!
```

This is a very trivial example of what Starcm can do. Let's make it a bit more complicated...

For instance, Starcm's `exec` can also handle non-zero exit codes.

### handling non-zero exit codes

See [examples/exec/exit_codes/unexpected.star](examples/exec/exit_codes/unexpected.star). 

<!-- Github Markdown engine will render this link as a code snippet. -->

https://github.com/discentem/starcm/blob/2911aea91ad6c978b94b1c237fe4fb38e69b32e2/examples/exec/exit_codes/unexpected.star#L1-L8

```scrut
$ bazel run :starcm -- --root_file examples/exec/exit_codes/unexpected.star --timestamps=false
we expect to exit 2
result(changed = True, diff = "", error = "exit status 2", name = "explicitly exit 2", output = "we expect to exit 2\n", success = False)
```

`exec` exited with a non-zero exit code thus `result` indicates things were not successful (`result(..., success=False)`). 

This is because the default `expected_error_code`, if not specified, is `0`.

>💡 What is `result()`? `result()` is a struct that is returned by most Starcm functions to signal whether a function achieved the expected result. Later we will see how Starcm code can consume the `result` struct to make conditional decisions.

If we set `expected_exit_code` to `2` then this succeeds!

<!-- Github Markdown engine will render this link as a code snippet. -->

https://github.com/discentem/starcm/blob/2911aea91ad6c978b94b1c237fe4fb38e69b32e2/examples/exec/exit_codes/expected.star#L1-L9


```scrut
$ bazel run :starcm -- --root_file examples/exec/exit_codes/expected.star --timestamps=false
we expect to exit 2
result(changed = True, diff = "", error = "exit status 2", name = "explicitly exit 2", output = "we expect to exit 2\n", success = True)
```

## rendering templates

Another thing Starcm can do is render template files via `template`. This is similar to the `template` resource in Chef. 

As an example let's take a look at [examples/templates/simple/template.star](examples/templates/simple/template.star).

https://github.com/discentem/starcm/blob/2911aea91ad6c978b94b1c237fe4fb38e69b32e2/examples/templates/simple/template.star#L1-L11

The template that is referenced in `template.star` is [examples/templates/simple/hello_world.tpl](examples/templates/simple/hello_world.tpl): 

https://github.com/discentem/starcm/blob/2911aea91ad6c978b94b1c237fe4fb38e69b32e2/examples/templates/simple/hello_world.tpl#L1-L1

```scrut
$ bazel run :starcm -- --root_file examples/templates/simple/template.star --timestamps=false -v 2
INFO: [LoadFromFile]: loading file "examples/templates/simple/template.star"
INFO: [hello world template]: hello_world.tpl before rendering: Hello {{ name | capitalize }}, you are {{ age }} years old.
INFO: [hello world template]: data: map[age:42 name:world]
INFO: [template]: skipping write to disk because persist is false
Hello World, you are 42 years old.
```


## Common functionality

All Starcm functions share some common functionality.

### `result`

All Starcm functions return a `result` struct. 

In Go this represented as such:

```go
type Result struct {
	Name    *string
	Output  *string
	Error   error
	Success bool
	Changed bool
	Diff    *string
	Comment string
}
```

If printed out or inspected directly in Starlark, a `result` may look something like this: 

```python
result(
    changed = True, 
    diff = "", 
    error = "exit status 2", 
    name = "explicitly exit 2", 
    output = "we expect to exit 2\n",   
    success = True
)
```

### Conditionals

<body>
<details>
<summary><h3 style="display:inline-block">if statements</h3></summary>

Starlark, and by extension starcm, supports `if` statements. Take [examples/if_statements/if_statements.star](examples/if_statements/if_statements.star) for example. If the `exec()` succeeds, we print `party!`. 

https://github.com/discentem/starcm/blob/2911aea91ad6c978b94b1c237fe4fb38e69b32e2/examples/if_statements/if_statements.star#L1-L11

```scrut
$ bazel run :starcm -- --root_file examples/if_statements/if_statements.star --timestamps=false
party!
```

We can also implement this same conditional behavior with a starcm-specific construct called `only_if`. This feature is not built into native Starlark.

</details>
</body>

<body>
<details>
<summary><h3 style="display:inline-block">only_if</h3></summary>

See [examples/only_if/only_if.star](examples/only_if/only_if.star).

https://github.com/discentem/starcm/blob/2911aea91ad6c978b94b1c237fe4fb38e69b32e2/examples/only_if/only_if.star#L1-L22

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

```scrut
$ bazel run :starcm -- --root_file examples/only_if/only_if.star --timestamps=false -v 2 
INFO: [LoadFromFile]: loading file "examples/only_if/only_if.star"
we expect to exit 2
INFO: [explicitly exit 2]: expectedExitCode: 2
INFO: [explicitly exit 2]: actualExitCode: 2
INFO: [print_not_success_#2]: skipping write(name="print_not_success_#2") because only_if was false
```

> Notice that there is no log message regarding `print_not_success_#1`. Normal `if` statements are not executed at all if the condition is false, whereas `only_if` logs that something was skipped.

</details>
</body>

# Advanced functionality

See the [examples](examples/) folder for more examples of what starcm can do. There's lots it can do such as downloading files (with hash checking), dynamically loading additional `.star` files, rendering templates, and combining all the cabilities via macros, thanks to Starlark.

# Starcm development

## Ensure README.md examples work

```shell
$ scrut test --work-directory . README.md
```