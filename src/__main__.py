# src/__main__.py

import asyncio
import os
import sys
import threading
import time
from typing import Sequence

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


def _has_cli_flag(flag: str, argv: Sequence[str] | None = None) -> bool:
    args = argv if argv is not None else sys.argv[1:]
    return flag in args


def _is_combined_server_scrape_mode(argv: Sequence[str] | None = None) -> bool:
    return _has_cli_flag('server', argv) and _has_cli_flag('scrape', argv)


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


def run_server(*, use_reloader: bool | None = None) -> None:
    allowed_origins = [
        r'https?://localhost(:\d+)?',
        r'https://estebmaister.github.io',
        # Tailscale magiclink and DNS
        r'https?://.*\.persian-nominal\.ts\.net(:\d+)?',
        r'https?://100\.103\.47\.96(:\d+)?',
    ]
    CORS(
        SERVER,
        resources={
            r'/comics.*': {'origins': allowed_origins},
            r'/scrape.*': {'origins': allowed_origins},
            r'/health.*': {'origins': '*'},
        }
    )

    # Production
    if PRODUCTION:
        http_server = WSGIServer(('0.0.0.0', PORT), SERVER)
        http_server.serve_forever()
        return
    # Development
    if use_reloader is None:
        use_reloader = DEBUG
    SERVER.run(host='0.0.0.0', port=PORT, debug=DEBUG,
               use_reloader=use_reloader,
               ssl_context=("./tls/comics.crt", "./tls/comics.key"))


def main(argv: Sequence[str] | None = None) -> None:
    if _is_combined_server_scrape_mode(argv):
        thread = threading.Thread(
            target=run_async_scrape,
            daemon=True,
            name='scrape-loop',
        )
        thread.start()
        # Flask's debug reloader forks a second process, which would start a
        # second scraper loop. Combined mode must stay single-process.
        run_server(use_reloader=False)
    elif _has_cli_flag('server', argv):
        run_server()
    else:
        scrapping()


if __name__ == '__main__':
    main()
