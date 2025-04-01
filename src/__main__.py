# src/__main__.py

import asyncio
import os
import sys
import threading
import time

from flask_cors import CORS
from gevent.pywsgi import WSGIServer

import helpers.logger
from scrape import scrapes
from server import server as SERVER

log = helpers.logger.get_logger(__name__)
PORT: int = int(os.getenv('PORT', 5001))
DEBUG: bool = os.getenv('DEBUG', 'false') == 'true'
PRODUCTION: bool = os.getenv('PRODUCTION', 'false') == 'true'

default_recurrence = 600  # 10 minutes


def run_async_scrape() -> None:
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    try:
        loop.run_until_complete(scrapping(10*default_recurrence))
    finally:
        # Close the loop
        loop.close()


def scrapping(recurrence: int = default_recurrence) -> None:
    scrape_cont = 1
    log.info('Scraping started...')
    while True:
        time_started = time.time()
        scrapes()

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

    # Production
    if PRODUCTION:
        http_server = WSGIServer(('0.0.0.0', PORT), SERVER)
        http_server.serve_forever()
    # Development
    SERVER.run(host='0.0.0.0', port=PORT, debug=DEBUG)


if 'server' in sys.argv and 'scrape' in sys.argv:
    thread = threading.Thread(target=run_async_scrape)
    thread.start()
    run_server()
elif 'server' in sys.argv:
    run_server()
else:
    scrapping()
