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

# query = osquery("SELECT serial FROM machines WHERE major_version = 15")'")

for serial in ["abc", "def"]: # for each serial in query
    sa = shard(
        name="santa shard for " + serial, 
        identifier=serial, 
        shard_size=10, 
        seed="santashard"
    )
    shardNum = sa.output

    chosen_env = choose_env(int(shardNum))

    url = "https://santa-{}.acme.com".format(chosen_env)
    render = template(
        name = "generating santa mobile config for serial '%s'" % serial, 
        template = 'santa.tmpl', 
        key_vals = {
           'SERVER_URL' : url
        }
    )
    write(render.output, name=("rendering output for serial choose_env(%s)" % shardNum))
