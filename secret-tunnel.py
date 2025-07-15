import sys
import yaml
import json
import math

secrets = []
for path in sys.argv[1:]:
    with open(path, "r") as f:
        content = yaml.safe_load(f.read())
        secrets.append({
            "name": content["nameOverride"],
            "kvpairs" : json.dumps(content["configmap"]["data"] | content["secret"]["data"]),
        })

out = yaml.dump({"secrets": secrets}, width=math.inf)
print(out)
