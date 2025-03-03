"""
Custom logger configuration for the comics application.

This module provides a consistent logging setup across the application,
with support for different log levels based on the DEBUG environment variable.
"""

import logging
import os
from typing import Optional

from dotenv import load_dotenv
from termcolor import colored

# Load environment variables
load_dotenv(override=True)

UPDATE = 31

logging._levelToName = {
    logging.CRITICAL: 'CRT',
    logging.ERROR: 'ERR',
    logging.WARNING: 'WRN',
    logging.INFO: 'INF',
    logging.DEBUG: 'DBG',
    UPDATE: 'UPD',
}

LOG_COLORS = {
    'DBG': 'blue',
    'INF': 'green',
    'WRN': 'yellow',
    'ERR': 'red',
    'CRT': 'magenta',
    'UPD': 'white'
}

# Configuration
DEBUG_MODE: bool = os.getenv('DEBUG', 'false').lower() == 'true'
LOG_LEVEL = logging.DEBUG if DEBUG_MODE else logging.INFO

# Log format configuration
LOG_FORMAT_PARTS = {
    'level': '[{levelname:s}]',  # Centered level name with fixed width
    # Only show timestamp in non-debug mode
    'timestamp': '{asctime:s}:' if not DEBUG_MODE else '',
    'module': '{name:^8s}:',  # Centered module name with fixed width
    'message': '{message:s}'  # The actual log message
}

# Combine format parts into final format string
LOG_FORMAT = ''.join(LOG_FORMAT_PARTS.values())


class ColoredFormatter(logging.Formatter):
    """Custom formatter that adds colors to log messages based on log level."""

    def format(self, record):
        msg = super().format(record)
        levelname = record.levelname
        color = LOG_COLORS.get(levelname, 'white')
        return colored(msg, color)


def configure_logging() -> None:
    """
    Configure the logging system with the appropriate format and level.
    Debug mode will show more detailed information and use DEBUG level.
    Non-debug mode will include timestamps and use INFO level.
    """
    if not DEBUG_MODE:
        logging.basicConfig(
            level=LOG_LEVEL,
            format=LOG_FORMAT,
            style='{'  # Use { style formatting for consistency with f-strings
        )
        return

    logger = logging.getLogger()
    logger.setLevel(LOG_LEVEL)

    # Create console handler, set level and a colored formatter
    formatter = ColoredFormatter(
        fmt=LOG_FORMAT,
        style='{'  # Use { style formatting for consistency with f-strings
    )
    console_handler = logging.StreamHandler()
    console_handler.setLevel(LOG_LEVEL)
    console_handler.setFormatter(formatter)

    # Clear any existing handlers to prevent duplicate log messages
    logger.handlers.clear()
    logger.addHandler(console_handler)


def get_logger(name: Optional[str] = None) -> logging.Logger:
    """
    Get a logger instance with the specified name.

    Args:
        name: The name of the logger, typically __name__ from the calling module.
            If None, returns the root logger.

    Returns:
        A configured logger instance that will use the application's format.
    """
    return logging.getLogger(name)


def update(message: str) -> None:
    """
    Logs a the update level, between INFO and ERROR.

    Args:
        message: The new log message to display.
    """
    logger = logging.getLogger(__name__)
    logger.log(UPDATE, message)


# Configure logging when module is imported
configure_logging()

# For backwards compatibility
logger = get_logger  # type: ignore
