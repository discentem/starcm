load("starcm", "load_dynamic")

for f in ['../echo/echo.star', '../nested_files/a.star']:
    load_dynamic(f)
