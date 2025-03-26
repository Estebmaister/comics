# src/__main__.py

import os
import sys
import time

from flask_cors import CORS
from gevent.pywsgi import WSGIServer

import helpers.logger
from db import session
from helpers.alert import send_reminder
from scrape import scrapes
from server import server as SERVER

log = helpers.logger.get_logger(__name__)
PORT: int = int(os.getenv('PORT', 5001))
DEBUG: bool = os.getenv('DEBUG', 'false') == 'true'

recurrence = 600  # 10 minutes


def scrapping() -> None:
    scrape_cont = 1
    log.info('Scraping started...')
    while True:
        time_started = time.time()
        scrapes(session)
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
                'https://estebmaister.github.io/*',
                # Tailscale magiclink and DNS
                r'http://.*\.persian-nominal\.ts\.net(:\d+)?',
                'http://100.103.47.96/*'
            ]},
            '/scrape': {'origins': [
                'http://localhost:*',
                'https://estebmaister.github.io/*',
                # Tailscale magiclink and DNS
                r'http://.*\.persian-nominal\.ts\.net(:\d+)?',
                'http://realme.persian-nominal.ts.net*',
                'http://100.103.47.96/*'
            ]},
            r'/health/*': {'origins': '*'},
        }
    )

    # Debug/Development
    if DEBUG:
        SERVER.run(host='0.0.0.0', port=PORT, debug=DEBUG)
    # Production
    http_server = WSGIServer(('', PORT), SERVER)
    http_server.serve_forever()


if 'server' in sys.argv:
    run_server()
else:
    scrapping()
