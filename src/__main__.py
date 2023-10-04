# src/__main__.py

import time, sys

from flask_cors import CORS
from scrape import scrapes
from server import server
from helpers.alert import reminder
import logging

DEBUG = False
logging.basicConfig(level=logging.INFO)
if 'debug' in sys.argv:
    # configure root logger
    logging.basicConfig(level=logging.DEBUG)
    logging.getLogger('flask_cors').level = logging.DEBUG
    DEBUG = True

recurrence = 600
def scrapping():
    scrap_cont = 1
    logging.info('Scraping started...')
    while True:
        time_started = time.time()
        scrapes()
        reminder()
        time_req = round((time.time() - time_started), 2)
        print(f'{scrap_cont} ({time_req})', end = '. ', flush=True)
        scrap_cont += 1
        time.sleep(recurrence)

def run_server():
    CORS( server, 
        resources={r'/comics/*': {'origins': ['http://localhost:3000']}}
    )
    server.run(port=5000, debug=DEBUG, host='localhost')

if 'server' in sys.argv:
    run_server()
else:
    scrapping()
