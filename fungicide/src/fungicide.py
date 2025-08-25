import json
import os
import re
from urllib.parse import urlparse
from dataclasses import dataclass, fields
from typing import cast, Any

from dotenv import load_dotenv

import redis
from redis.backoff import ExponentialBackoff
from redis.retry import Retry

import joblib
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.linear_model import LogisticRegression


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
    input_queue_key: str
    output_queue_key: str
    blacklist_key: str

    # webpage classifier
    clf: LogisticRegression
    vectorizer: TfidfVectorizer
    rejection_threshold: float

    def wait_for_page(self, timeout=0) -> Page:
        raw = self.redis_client.blpop([self.input_queue_key], timeout)
        _, value = cast(tuple[str, bytes], raw)
        json_str = value.decode(encoding='utf-8')
        return json.loads(json_str, object_hook = Page.as_page)

    def push_page(self, page: Page):
        self.redis_client.rpush(self.output_queue_key, json.dumps(page))

    def blacklist(self, page: Page):
        domain = urlparse(page.location).netloc
        self.redis_client.sadd(self.blacklist_key, domain)

    def classify(self, page: Page) -> tuple[float, float]:
        text = page.tokenize()
        matrix = self.vectorizer.transform([text])
        not_dev, dev = self.clf.predict_proba(matrix)
        return not_dev, dev


def main():
    load_dotenv()
    app = init_app()

    print("app started. waiting for input...")

    while True:
        page = app.wait_for_page()
        not_dev_proba, _ = app.classify(page)

        print(not_dev_proba, page.location)

        if not_dev_proba < app.rejection_threshold:
            app.push_page(page)
        else:
            app.blacklist(page)


def init_app() -> App:
    redis_host          = os.getenv('REDIS_HOST', '')
    redis_port          = os.getenv('REDIS_PORT', '')
    redis_max_retries   = os.getenv('REDIS_MAX_RETRIES', '')
    input_queue_key     = os.getenv('REDIS_INPUT_QUEUE_KEY', '')
    output_queue_key    = os.getenv('REDIS_OUTPUT_QUEUE_KEY', '')
    blacklist_key       = os.getenv('REDIS_BLACKLIST_KEY', '')
    model_file          = os.getenv('MODEL_FILE', '')
    vectorizer_file     = os.getenv('VECTORIZER_FILE', '')
    rejection_threshold = os.getenv('REJECTION_THRESHOLD', '')

    retry = Retry(ExponentialBackoff(), int(redis_max_retries))
    client = redis.Redis(
        host=redis_host,
        port=int(redis_port),
        decode_responses=True,
        retry=retry,
    )

    clf, vectorizer = None, None
    if os.path.isfile(model_file) and os.path.isfile(vectorizer_file):
        clf, vectorizer = joblib.load(model_file), joblib.load(vectorizer_file)
    else:
        raise FileNotFoundError("failed to load model or vectorizer")

    app = App(
        redis_client=client,
        input_queue_key=input_queue_key,
        output_queue_key=output_queue_key,
        blacklist_key=blacklist_key,
        clf=clf,
        vectorizer=vectorizer,
        rejection_threshold=int(rejection_threshold) / 100.0,
    )

    print("successfully initialized app")
    return app


if __name__ == "__main__":
    main()
