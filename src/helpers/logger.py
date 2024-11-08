import logging
import os
from dotenv import load_dotenv

load_dotenv(override=True)
DEBUG: bool = os.getenv('DEBUG', 'false') == 'true'

# Longest logging level: CRITICAL (8)
format='[{levelname:^8s}]'
if not DEBUG:
  format = format + '{asctime:s}:'
format = format + '{name:^8s}:{message:s}'
style='{'

if DEBUG:
  logging.basicConfig(level=logging.DEBUG, format=format, style=style)
else:
  logging.basicConfig(level=logging.INFO , format=format, style=style)

def logger(name=None):
  return logging.getLogger(name)