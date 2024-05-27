# starcm

- A rudimentary config management tool that utilizes Starlark as the configuration language instead of json or yaml. 
- This is not full replacment for tools like Chef or Ansible. But it can be used to bootstrap them!

# Usage

`go run main.go -v 2 --root_file examples/example.star`


# Custom Starlark Functions

## exec

Example:
```python
exec(
    cmd  = "echo", 
    args = ["hello"],
)
```