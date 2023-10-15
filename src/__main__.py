# src/__main__.py

import logging
import time, sys, os
from flask_cors import CORS
from gevent.pywsgi import WSGIServer
from scrape import scrapes
from server import server as SERVER
from helpers.alert import reminder
from dotenv import load_dotenv

load_dotenv()
PORT: int = os.getenv('PORT', 5000)
DEBUG: bool = os.getenv('DEBUG', False)

logging.basicConfig(level=logging.INFO)
if DEBUG:
    # configure root logger
    logging.basicConfig( level=logging.DEBUG )
    logging.getLogger( 'flask_cors' ).level = logging.DEBUG

recurrence = 600 # 10 minutes
def scrapping() -> None:
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

def run_server() -> None:
    CORS( SERVER, 
        resources={
            r'/comics/*': {'origins': [
                'http://localhost:3000',
                'https://estebmaister.github.io/*'
            ]}, 
            '/health/': {'origins':'*'}}
    )
    
    # Debug/Development
    if DEBUG: SERVER.run(host='', port=PORT, debug=DEBUG)
    # Production
    http_server = WSGIServer(('', PORT), SERVER)
    http_server.serve_forever()

if 'server' in sys.argv:
    run_server()
else:
    scrapping()
