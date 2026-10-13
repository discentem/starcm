load("starcm", "json")

d = json.decode('{"name": "ada", "tags": ["a", "b"], "n": 3, "ok": true, "none": null}')
print(d["name"], d["tags"], d["n"] + 1, d["ok"], d["none"])
print(json.encode({"b": [1, None], "a": True}))
print(json.indent(json.encode({"a": 1})))
