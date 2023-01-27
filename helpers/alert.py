#!/usr/bin/env python

import subprocess
from typing import List
from db.models import Publishers

default_content = "Update found"
msg = dict( title = "Scrap alert", content = default_content, alert = 0 )

def reminder():
    if msg["alert"] == 0:
        return
    subprocess.Popen([ 'notify-send',
        msg.get("title") + f" - ({ msg.get('alert') })", 
        msg.get("content")])
    msg["alert"] = 0
    msg["content"] = default_content

def add_alert_to_msg(title: str, chap: str, publisher: List[Publishers]):
    msg["alert"] += 1
    msg["content"] += f"\t\n{title}, ch {chap}"
    msg["content"] += f" - {[Publishers(pub).name for pub in publisher]}"
    if msg["alert"] == 4:
        reminder()