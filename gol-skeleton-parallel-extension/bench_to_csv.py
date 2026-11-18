import csv
import re
import sys
from pathlib import Path

log = Path(sys.argv[1] if len(sys.argv) > 1 else "bench.log")
data = log.read_bytes()

text = ""
for enc in ("utf-8-sig", "utf-16", "gbk"):
    try:
        text = data.decode(enc)
        break
    except UnicodeDecodeError:
        continue

pattern = re.compile(
    r"BenchmarkMapReduce/(\d+)x(\d+)x(\d+)-(\d+)-\d+\s+\d+\s+([0-9.]+) ns/op"
)

rows = []
for line in text.splitlines():
    m = pattern.search(line)
    if m:
        width, height, turns, threads, ns = m.groups()
        rows.append(["mapreduce", width, threads, round(float(ns) / int(turns))])

with open("results.csv", "w", newline="", encoding="utf-8") as f:
    writer = csv.writer(f)
    writer.writerow(["engine", "size", "threads", "ns_per_turn"])
    writer.writerows(rows)

print(f"{len(rows)} rows -> results.csv")