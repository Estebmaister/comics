import json
import os

url_switch = {}
url_switch_file = os.path.join(os.path.dirname(__file__), "url_switch.json")
with open(url_switch_file) as js_file:
    js_read_file = js_file.read()
    url_switch = json.loads(js_read_file)

publisher_url_pairs = []
for pub in url_switch.keys():
    for url in url_switch.get(pub):
        publisher_url_pairs.append((pub, url))
