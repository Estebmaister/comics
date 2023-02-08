# src/__main__.py

import time, signal, sys
from scrap import scraps
from server import server
from helpers.alert import reminder
from db import load_comics, session, save_comics_file

def signal_handler(sig, frame):
    session.close()
    save_comics_file(load_comics)
    print(' Closing gracefully...')
    sys.exit(0)

signal.signal(signal.SIGINT, signal_handler)

recurrence = 600
def scrapping():
    scrap_cont = 1
    print('Scraping...')
    print('.', end = '')
    while True:
        scraps(load_comics)
        save_comics_file(load_comics)
        reminder()
        print(str(scrap_cont) + '.', end = '')
        scrap_cont += 1
        time.sleep(recurrence)

def run_server():
    server.run(port=5000, debug=True, host='localhost')

if 'server' in sys.argv:
    run_server()
else:
    scrapping()
