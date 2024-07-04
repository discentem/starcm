load("template", "template")
load("write", "write")
load("shard", "shard")

def choose_env(shard):
    if shard < 4:
        return "dev"
    elif shard < 8:
        return "staging"
    else:
        return "prod"


for serial in ["abc", "def"]:
    sa = shard(
        name="santa shard for " + serial, 
        identifier=serial, 
        shard_size=10, 
        seed="santashard"
    )
    chosen_env = choose_env(int(sa.output))

    url = "https://santa-{}.acme.com".format(chosen_env)
    render = template(
        name = 'generating santa mobile config', 
        template = 'santa.tmpl', 
        key_vals = {
           'SERVER_URL' : url
        }
    )
    write(render.output)
