load("starcm", "exec")

exec(label = "installer", cmd = "sh", args = ["-c", "echo installing; exit 3"], check = True)
print("this line never runs")
