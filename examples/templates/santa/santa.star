load("template", "template")
load("write", "write")

for serial in [
    "fdsafAD114e324"
]:
    render = template(
        name = "generating santa mobile config",
        template = "santa.tpl",
        key_vals = {
           "server_url": "https://santa-%s.example.com" % int(serial, 16),
        }
    )
    write(render.output)
