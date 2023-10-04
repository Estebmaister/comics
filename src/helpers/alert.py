# python helpers/alert.py alert

import subprocess, sys
from typing import List
from db import Publishers

default_content = "Update found"
msg = dict( title = "Scrape alert", content = default_content, alert = 0 )
alert_icon = "/usr/share/icons/Adwaita/scalable/"
alert_icon += "status/software-update-urgent-symbolic.svg"

def reminder(_send: bool = False):
    if msg["alert"] == 0 and not _send:
        return
    subprocess.Popen([ 'notify-send', "-i", alert_icon, "-u", "critical",
        msg.get("title") + f" - ({ msg.get('alert') })", 
        msg.get("content")])
    msg["alert"] = 0
    msg["content"] = default_content

def add_alert_to_msg(title: str, chap: str, publisher: List[Publishers]):
    publishers_to_look = f"- {[Publishers(pub).name for pub in publisher]}"
    update_msg = f"\t\n{title}, ch <b>{chap}</b> {publishers_to_look}"
    print(title,chap,publishers_to_look)
    msg["alert"] += 1
    msg["content"] += update_msg
    if msg["alert"] == 4:
        reminder()

if 'alert' in sys.argv:
    reminder(True)