# src/__main__.py

import os
import sys
import time

from flask_cors import CORS
from gevent.pywsgi import WSGIServer

from helpers.alert import send_reminder
from helpers.logger import logger
from scrape import scrapes
from server import server as SERVER

log = logger(__name__)
PORT: int = int(os.getenv('PORT', 5001))
DEBUG: bool = os.getenv('DEBUG', 'false') == 'true'

recurrence = 600  # 10 minutes


def scrapping() -> None:
    scrape_cont = 1
    log.info('Scraping started...')
    while True:
        time_started = time.time()
        scrapes()
        send_reminder()
        time_req = round((time.time() - time_started), 2)
        print(f'{scrape_cont} ({time_req})', end='\n', flush=True)
        scrape_cont += 1
        time.sleep(recurrence)


def run_server() -> None:
    CORS(
        SERVER,
        resources={
            r'/comics/*': {'origins': [
                'http://localhost:*',
                'https://estebmaister.github.io/*'
            ]},
            '/scrape': {'origins': [
                'http://localhost:*',
                'https://estebmaister.github.io/*'
            ]},
            r'/health/*': {'origins': '*'},
        }
    )

    # Debug/Development
    if DEBUG:
        SERVER.run(host='', port=PORT, debug=DEBUG)
    # Production
    http_server = WSGIServer(('', PORT), SERVER)
    http_server.serve_forever()


if 'server' in sys.argv:
    run_server()
else:
    scrapping()
