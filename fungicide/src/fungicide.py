import json
import os
import re
from urllib.parse import urlparse
from dataclasses import dataclass, is_dataclass, asdict, fields
from typing import cast, Any

from dotenv import load_dotenv

import redis
from redis.backoff import ExponentialBackoff
from redis.retry import Retry

import joblib
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.linear_model import LogisticRegression


class EnhancedJSONEncoder(json.JSONEncoder):
    def default(self, o: object):
        if is_dataclass(o):
            return asdict(o)
        return super().default(o)


@dataclass
class Outlink:
    location: str
    retries:  int

    @staticmethod
    def as_outlink(map: dict[str, Any]) -> 'Outlink':
        """
        Convert a dictionary into a page object.
        """
        return Outlink(
            location=map.get("location", ""),
            retries=map.get("retries", "")
        )


@dataclass
class Page(object):
    title:          str
    description:    str
    author:         str
    keywords:       list[str]
    headings:       list[str]
    content:        list[str]
    links:          list[str]
    script_links:   list[str]
    script_content: list[str]
    location:       str

    def tokenize(self):
        """
        Tokenize page content into a single string.
        """
        text_to_tokenize = []
        for field in fields(self):
            value = getattr(self, field.name)
            if value:
                if isinstance(value, list):
                    text_to_tokenize.extend(value)
                elif isinstance(value, str):
                    text_to_tokenize.append(value)

        combined_text = " ".join(text_to_tokenize)
        raw = re.findall(r'\b\w+\b', combined_text.lower())
        tokens = [t for t in raw if all(c.isascii() and c.isalpha() for c in t) and len(t) > 3]
        return " ".join(tokens)

    @staticmethod
    def as_page(map: dict[str, Any]) -> 'Page':
        """
        Convert a dictionary into a page object.
        """
        return Page(
            title=map.get("title", ""),
            description=map.get("description", ""),
            author=map.get("author", ""),
            keywords=map.get("keywords", []),
            headings=map.get("headings", []),
            content=map.get("content", []),
            links=map.get("links", []),
            script_links=map.get("script_links", []),
            script_content=map.get("script_content", []),
            location=map.get("location", ""),
        )


@dataclass
class App:
    # input/output queues
    redis_client: redis.Redis
    fungicide_queue_key: str
    taxonomist_queue_key: str
    mycelium_queue_key: str
    mycelium_blacklist_key: str

    # webpage classifier
    clf: LogisticRegression
    vectorizer: TfidfVectorizer
    rejection_threshold: float

    def run(self):
        """
        Continually poll the redis queue for pages and process them.
        """
        while True:
            page = self.wait_for_page()
            not_dev_proba, _ = self.classify(page)

            if not_dev_proba >= self.rejection_threshold:
                self.blacklist(page)
                print("BLOCK", int(not_dev_proba * 100), page.location)
            elif not_dev_proba < 50:
                self.push_page(page)
                self.push_outlinks(page)
                print("PUSH1", int(not_dev_proba * 100), page.location)
            else:
                self.push_outlinks(page)
                print("PUSH2", int(not_dev_proba * 100), page.location)

    def wait_for_page(self, timeout=0) -> Page:
        """
        Wait for a page from the redis queue.
        """
        raw = self.redis_client.blpop([self.fungicide_queue_key], timeout)
        _, value = cast(tuple[str, str], raw)
        return json.loads(value, object_hook = Page.as_page)

    def push_outlinks(self, page: Page):
        """
        Also push the page outlinks to the crawler's ingest queue.
        """
        s_to_outlink = lambda s: json.dumps(Outlink(location=s, retries=0),
                                            cls=EnhancedJSONEncoder)
        outlinks = [s_to_outlink(link) for link in (page.links or [])]

        if len(outlinks) > 0:
            self.redis_client.rpush(self.mycelium_queue_key, *outlinks)

    def push_page(self, page: Page):
        """
        Push a page to the redis output queue.
        """
        self.redis_client.rpush(self.taxonomist_queue_key, json.dumps(page, cls=EnhancedJSONEncoder))

    def blacklist(self, page: Page):
        """
        Add a page's domain to the redis blacklist.
        """
        domain = urlparse(page.location).netloc
        self.redis_client.sadd(self.mycelium_blacklist_key, domain)

    def classify(self, page: Page) -> tuple[float, float]:
        """
        Classify a page. Returns confidence from 0-1 if page is not a dev blog
        and if a page is a dev blog respectively.
        """
        text = page.tokenize()
        matrix = self.vectorizer.transform([text])
        return self.clf.predict_proba(matrix)[0]


def init_app() -> App:
    """
    Read configurations from environment and instantiate app.
    """
    load_dotenv()
    redis_host             = os.getenv('REDIS_HOST', '')
    redis_port             = os.getenv('REDIS_PORT', '')
    redis_max_retries      = os.getenv('REDIS_MAX_RETRIES', '')
    fungicide_queue_key    = os.getenv('REDIS_FUNGICIDE_QUEUE_KEY', '')
    taxnonomist_queue_key  = os.getenv('REDIS_TAXONOMIST_QUEUE_KEY', '')
    mycelium_queue_key     = os.getenv('REDIS_MYCELIUM_QUEUE_KEY', '')
    mycelium_blacklist_key = os.getenv('REDIS_MYCELIUM_BLACKLIST_KEY', '')
    model_file             = os.getenv('MODEL_FILE', '')
    vectorizer_file        = os.getenv('VECTORIZER_FILE', '')
    rejection_threshold    = os.getenv('REJECTION_THRESHOLD', '')

    # create redis client
    retry = Retry(ExponentialBackoff(), int(redis_max_retries))
    client = redis.Redis(
        host=redis_host,
        port=int(redis_port),
        decode_responses=True,
        retry=retry,
    )

    # load model and vectorizer
    clf, vectorizer = None, None
    if os.path.isfile(model_file) and os.path.isfile(vectorizer_file):
        clf, vectorizer = joblib.load(model_file), joblib.load(vectorizer_file)
    else:
        raise FileNotFoundError("failed to load model or vectorizer")

    app = App(
        redis_client=client,
        fungicide_queue_key=fungicide_queue_key,
        taxonomist_queue_key=taxnonomist_queue_key,
        mycelium_queue_key=mycelium_queue_key,
        mycelium_blacklist_key=mycelium_blacklist_key,
        clf=clf,
        vectorizer=vectorizer,
        rejection_threshold=int(rejection_threshold) / 100.0,
    )

    print("successfully initialized app")
    return app


def main():
    app = init_app()
    print("app started. waiting for input...")
    app.run()


if __name__ == "__main__":
    main()
