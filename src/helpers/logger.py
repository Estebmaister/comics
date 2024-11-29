"""
Custom logger configuration for the comics application.

This module provides a consistent logging setup across the application,
with support for different log levels based on the DEBUG environment variable.
"""

import logging
import os
from typing import Optional

from dotenv import load_dotenv

# Load environment variables
load_dotenv(override=True)

# Configuration
DEBUG_MODE: bool = os.getenv('DEBUG', 'false').lower() == 'true'
LOG_LEVEL = logging.DEBUG if DEBUG_MODE else logging.INFO

# Log format configuration
LOG_FORMAT_PARTS = {
    'level': '[{levelname:^8s}]',  # Centered level name with fixed width
    # Only show timestamp in non-debug mode
    'timestamp': '{asctime:s}:' if not DEBUG_MODE else '',
    'module': '{name:^8s}:',  # Centered module name with fixed width
    'message': '{message:s}'  # The actual log message
}

# Combine format parts into final format string
LOG_FORMAT = ''.join(LOG_FORMAT_PARTS.values())


def configure_logging() -> None:
    """
    Configure the logging system with the appropriate format and level.
    Debug mode will show more detailed information and use DEBUG level.
    Non-debug mode will include timestamps and use INFO level.
    """
    logging.basicConfig(
        level=LOG_LEVEL,
        format=LOG_FORMAT,
        style='{'  # Use { style formatting for consistency with f-strings
    )


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


# Configure logging when module is imported
configure_logging()

# For backward compatibility
logger = get_logger  # type: ignore
