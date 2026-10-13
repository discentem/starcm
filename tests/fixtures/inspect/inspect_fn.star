load("starcm", "inspect")

r = inspect("main.star", label = "inspect main.star").value
print([c["function"] for c in r["calls"]])
print([c.get("path") for c in r["calls"] if c["function"] == "load_dynamic"])
print(r["loads"][0]["names"]["run"])
