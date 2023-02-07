## Scraping webs to shw latest comics

from db.models import load_comics, session, save_comics_file
from helpers.alert import reminder
from src.scraps import scraps
from server.server import server
import json, time, signal, sys

def signal_handler(sig, frame):
    session.close()
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
    if __name__ == '__main__':
        server.run(port=5000,debug=True, host='localhost')

if 'server' in sys.argv:
    run_server()
else:
    scrapping()
