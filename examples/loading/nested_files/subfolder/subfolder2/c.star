print("hello from c.star")

load("starcm", "load_dynamic")

load_dynamic("subfolder3/d.star", label="load d.star from c.star")

load_dynamic("//examples/loading/nested_files/e.star", label="load e.star from c.star")
