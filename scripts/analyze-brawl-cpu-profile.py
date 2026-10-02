"""Summarize sampled CPU time by call frame; no claim about GPU time."""
import json
import sys
from collections import defaultdict
from pathlib import Path

source = Path(sys.argv[1])
profile = json.loads(source.read_text(encoding="utf-8"))
nodes = {node["id"]: node for node in profile["nodes"]}
parents = {child: node["id"] for node in profile["nodes"] for child in node.get("children", [])}
self_us, total_us = defaultdict(int), defaultdict(int)
samples, deltas = profile["samples"], profile["timeDeltas"]
if len(samples) != len(deltas):
    raise ValueError("sample/duration count mismatch")
for identifier, duration in zip(samples, deltas):
    self_us[identifier] += duration
    while identifier is not None:
        total_us[identifier] += duration
        identifier = parents.get(identifier)

def describe(identifier, duration):
    frame = nodes[identifier]["callFrame"]
    return {"function": frame["functionName"], "url": frame["url"], "line": frame["lineNumber"] + 1, "ms": round(duration / 1000, 3)}

result = {"source": source.name, "samples": len(samples), "sampledMS": sum(deltas)/1000,
          "self": [describe(i, d) for i, d in sorted(self_us.items(), key=lambda row: -row[1])[:30]],
          "inclusive": [describe(i, d) for i, d in sorted(total_us.items(), key=lambda row: -row[1])[:30]]}
output = source.with_suffix(".summary.json")
output.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
print(json.dumps(result, indent=2))
