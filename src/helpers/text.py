import re
import unicodedata

SMART_QUOTES = {
    "\u2018": "'",
    "\u2019": "'",
    "\u201c": '"',
    "\u201d": '"',
    "\u2013": "-",
    "\u2014": "-",
}


def normalize_text(value: str) -> str:
    if value is None:
        return ""
    text = unicodedata.normalize("NFKC", str(value))
    for src, dst in SMART_QUOTES.items():
        text = text.replace(src, dst)
    text = re.sub(r"\s+", " ", text)
    return text.strip()


def normalize_title_list(titles):
    if isinstance(titles, list):
        return [normalize_text(title) for title in titles]
    if isinstance(titles, str):
        return normalize_text(titles)
    return titles
