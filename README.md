# starcm

A rudimentary config management tool that utilizes Starlark. This is not full replacment for cm tools like Chef or Ansible but it is useful for bootstrapping those tools!

# Usage

`go run main.go example.star`

You can also write your own .star file and execute it!

`go run main.go newthing.star`

# Custom Starlark Functions

## exec

Example:
```python
exec(
    cmd  = "echo", 
    args = ["hello"],
)
```