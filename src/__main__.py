# src/__main__.py

import logging
import time, sys, os
from flask_cors import CORS
from gevent.pywsgi import WSGIServer
from scrape import scrapes
from server import server
from helpers.alert import reminder

DEBUG = False
logging.basicConfig(level=logging.INFO)
if 'debug' in sys.argv:
    # configure root logger
    logging.basicConfig(level=logging.DEBUG)
    logging.getLogger('flask_cors').level = logging.DEBUG
    DEBUG = True

recurrence = 600
def scrapping():
    scrape_cont = 1
    logging.info('Scraping started...')
    while True:
        time_started = time.time()
        scrapes()
        reminder()
        time_req = round((time.time() - time_started), 2)
        print(f'{scrape_cont} ({time_req})', end = '. ', flush=True)
        scrape_cont += 1
        time.sleep(recurrence)

def run_server():
    CORS( server, 
        resources={r'/comics/*': {'origins': ['http://localhost:3000']}}
    )
    port = int(os.getenv('PORT', 5000))
    # Debug/Development
    if DEBUG: server.run(port=port, host='localhost')
    # Production
    http_server = WSGIServer(('', port), server)
    http_server.serve_forever()

if 'server' in sys.argv:
    run_server()
else:
    scrapping()
