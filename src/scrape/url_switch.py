"""
Module for managing publisher URL mappings.

This module loads and provides access to URL mappings for different comic publishers
from a JSON configuration file.
"""

import json
import os
from typing import List, Tuple

# Load URL mappings from configuration file
url_switch = {}
url_switch_file = os.path.join(os.path.dirname(__file__), "url_switch.json")

def load_url_mappings() -> dict:
    """Load publisher URL mappings from the configuration file."""
    with open(url_switch_file) as js_file:
        return json.loads(js_file.read())

def get_publisher_url_pairs() -> List[Tuple[str, str]]:
    """
    Generate pairs of publisher and URL combinations.
    
    Returns:
        List of tuples containing (publisher, url) pairs.
    """
    pairs = []
    for publisher in url_switch.keys():
        for url in url_switch.get(publisher, []):
            pairs.append((publisher, url))
    return pairs

# Initialize the URL mappings
url_switch = load_url_mappings()
publisher_url_pairs = get_publisher_url_pairs()
