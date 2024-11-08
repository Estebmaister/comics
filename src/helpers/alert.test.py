import subprocess

default_content = 'Update not found'
msg = dict( title = 'Scrape alert', content = default_content, alert = 0 )
alert_icon = '/usr/share/icons/Adwaita/scalable/'
alert_icon += 'status/software-update-urgent-symbolic.svg'

def reminder(_send: bool = False):
    if msg['alert'] == 0 and not _send:
        return
    try:
        subprocess.Popen([ 'notify-send',
            msg.get('title') + f' - ({ msg.get('alert') })', 
            msg.get('content')])
    except FileNotFoundError:
        print('MSG:', 'Notifier not found - ', msg.get('content'))
    msg['alert'] = 0
    msg['content'] = default_content

if __name__ == '__main__':
    reminder(True)