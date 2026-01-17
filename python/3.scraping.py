import sys
from concurrent.futures import ThreadPoolExecutor
from bs4 import BeautifulSoup
import requests

urls = [
    "https://www.python.org/doc/",
    "https://golang.org/doc/",
    "https://docs.djangoproject.com/en/stable/",
    "https://flask.palletsprojects.com/en/stable/",
    "https://fastapi.tiangolo.com/",
    "https://pandas.pydata.org/docs/",
    "https://numpy.org/doc/",
    "https://scikit-learn.org/stable/documentation.html",
    "https://matplotlib.org/stable/contents.html",
    "https://developer.mozilla.org/en-US/docs/Web",
    "https://news.ycombinator.com/",
    "https://www.theguardian.com/international",
    "https://www.reuters.com/",
    "https://www.cnn.com/world",
    "https://www.nytimes.com/international/",
]

HEADERS = {"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"}
TIMEOUT = 30

def fetch(url):
    """Fetch URL and parse HTML - runs in separate thread"""
    try:
        r = requests.get(url, headers=HEADERS, timeout=TIMEOUT)
        return BeautifulSoup(r.text, "html.parser").get_text()
    except Exception as e:
        print(f"Failed to fetch {url}: {e}")
        return ""

def fetch_urls_threaded(max_workers=None):
    """Fetch all URLs using thread pool (true parallelism with no-GIL Python)"""
    if max_workers is None:
        max_workers = len(urls)

    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        results = list(executor.map(fetch, urls))
    return results

def main():
    html_pages = fetch_urls_threaded()

if __name__ == "__main__":
    main()
