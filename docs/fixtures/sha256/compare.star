load("starcm", "sha256", "write")

EXPECTED = "2cddd2be80b2292ca06a9c962200f04a823cb5f614031951da92c2ffeba87e7d"
actual = sha256(path = "payload.txt", label = "hash payload").value
write("payload is intact", label = "ok", only_if = actual == EXPECTED)
write("payload changed!", label = "changed", not_if = actual == EXPECTED)
