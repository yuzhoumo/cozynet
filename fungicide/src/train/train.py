import os
import re
import json
import joblib
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.linear_model import LogisticRegression
from sklearn.model_selection import train_test_split
from sklearn.metrics import classification_report
from sklearn.feature_extraction import text


def tokenize(text):
    raw_tokens = re.findall(r'\b\w+\b', text.lower())
    return [t for t in raw_tokens if all(c.isascii() and c.isalpha() for c in t) and len(t) > 3]


def process_json_file(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    text_to_tokenize = []
    for field in ['title', 'description', 'keywords', 'headings', 'content']:
        if field in data:
            if isinstance(data[field], list):
                text_to_tokenize.extend(data[field])
            elif isinstance(data[field], str):
                text_to_tokenize.append(data[field])
    combined_text = " ".join(text_to_tokenize)
    return " ".join(tokenize(combined_text))


def load_dataset(folder, label):
    texts, labels = [], []
    for domain_folder in os.listdir(folder):
        domain_path = os.path.join(folder, domain_folder)
        if not os.path.isdir(domain_path):
            continue
        for file_name in os.listdir(domain_path):
            if file_name.endswith(".json"):
                file_path = os.path.join(domain_path, file_name)
                text = process_json_file(file_path)
                if text.strip():  # only keep non-empty
                    texts.append(text)
                    labels.append(label)
    return texts, labels


def train(dev_dir, nondev_dir):
    dev_texts, dev_labels = load_dataset(dev_dir, 1)
    nondev_texts, nondev_labels = load_dataset(nondev_dir, 0)

    texts = dev_texts + nondev_texts
    labels = dev_labels + nondev_labels

    # Vectorize
    vectorizer = TfidfVectorizer(
        max_features=10000,
        ngram_range=(1,3),
        min_df=2,   # ignore words appearing in only 1 doc
        max_df=0.9, # ignore words appearing in 90% of docs
        stop_words=list(text.ENGLISH_STOP_WORDS),
    )

    X = vectorizer.fit_transform(texts)
    y = labels

    # Train/test split
    X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)

    # Train classifier
    clf = LogisticRegression(max_iter=2000, class_weight='balanced')
    clf.fit(X_train, y_train)

    # Evaluate
    y_pred = clf.predict(X_test)
    print(classification_report(y_test, y_pred))

    return clf, vectorizer


def classify_json(file_path, clf, vectorizer):
    text = process_json_file(file_path)
    X_new = vectorizer.transform([text])
    return clf.predict_proba(X_new)[0] # [prob_not_dev, prob_dev]


if __name__ == '__main__':
    model_file, vectorizer_file = '../models/model.dump', "../models/vectorizer.dump"
    if os.path.isfile(model_file) and os.path.isfile(vectorizer_file):
        clf = joblib.load(model_file)
        vectorizer = joblib.load(vectorizer_file)
    else:
        clf, vectorizer = train("./datasets/positive", "./datasets/negative")
        joblib.dump(clf, model_file)
        joblib.dump(vectorizer, vectorizer_file)

    print(classify_json("./out.json", clf, vectorizer))
