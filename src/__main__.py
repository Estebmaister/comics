# src/__main__.py

import os
import sys
from typing import Sequence

from gevent.pywsgi import WSGIServer

import helpers.logger
from server import server as SERVER

log = helpers.logger.get_logger(__name__)
PORT: int = int(os.getenv('PORT', 5001))
DEBUG: bool = os.getenv('DEBUG', 'false') == 'true'
PRODUCTION: bool = os.getenv('PRODUCTION', 'false') == 'true'


def _has_cli_flag(flag: str, argv: Sequence[str] | None = None) -> bool:
    args = argv if argv is not None else sys.argv[1:]
    return flag in args


def run_server(*, use_reloader: bool | None = None) -> None:
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
    if _has_cli_flag('server', argv):
        run_server()
        return
    if _has_cli_flag('scrape', argv):
        log.error(
            'Python scrape CLI removed; use Go: make go-scrape '
            '(or enable SCRAPE_INTERVAL on make go-run)'
        )
        sys.exit(2)
        return
    log.error(
        'Unknown entrypoint. Use: python3 src server  '
        '(scraping: make go-scrape / Go GET /scrape)'
    )
    sys.exit(2)


if __name__ == '__main__':
    main()
