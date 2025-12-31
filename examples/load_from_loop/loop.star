load("starcm", "load_dynamic")

for f in ['../nested_files/e.star', '../nested_files/subfolder/subfolder2/subfolder3/d.star']:
    load_dynamic(f, label=f)
