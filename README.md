# Usage

`go run main.go example.star`

You can also write your own .star file and execute it!

`go run main.go whatever.star`

# Custom Starlark Functions

## exec

Example:
```python
exec(
    cmd  = "echo", 
    args = ["hello"],
)
```