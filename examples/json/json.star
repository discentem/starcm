load("starcm", "json")

doc = json.decode('{"name": "ada", "groups": ["admin", "dev"], "uid": 501, "active": true, "shell": null}')
print(doc["name"], doc["groups"], doc["uid"] + 1, doc["active"], doc["shell"])

print(json.encode({"b": [1, None], "a": True}))
print(json.indent('{"a":1,"b":[1,2]}', indent = "  "))
