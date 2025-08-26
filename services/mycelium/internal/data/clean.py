import concurrent.futures
import urllib.request
import urllib.error
import ssl
import socket
from urllib.parse import urlparse
from pathlib import Path

SEED_FILE = Path("seed.txt")
WORKING_FILE = Path("seed_working.txt")
TIMEOUT = 5
MAX_WORKERS = 50

BROWSER_UA = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
    "AppleWebKit/537.36 (KHTML, like Gecko) "
    "Chrome/115.0 Safari/537.36"
)

def try_url(url):
    """Attempt to open URL; return (success_url, error_str)."""
    try:
        req = urllib.request.Request(url, headers={"User-Agent": BROWSER_UA}, method="GET")
        context = ssl.create_default_context()
        context.check_hostname = True
        with urllib.request.urlopen(req, timeout=TIMEOUT, context=context):
            return url, None
    except urllib.error.HTTPError as e:
        return None, f"HTTP {e.code}"
    except urllib.error.URLError as e:
        return None, f"URL Error: {e.reason}"
    except ssl.SSLError as e:
        return None, f"SSL Error: {e}"
    except socket.timeout:
        return None, "Timeout"
    except Exception as e:
        return None, f"{type(e).__name__}: {e}"

def generate_fallbacks(original):
    """Yield fallback URLs to try, in order."""
    parsed = urlparse(original if "://" in original else "http://" + original)

    schemes = ["https", "http"] if parsed.scheme == "https" else ["http", "https"]

    # Host variants
    host_variants = []
    if parsed.hostname:
        host_variants.append(parsed.hostname)

    if parsed.hostname and parsed.path not in ("", "/"):
        host_variants.append(parsed.hostname)  # Root host

    # Remove www. or subdomain to get base domain
    if parsed.hostname:
        parts = parsed.hostname.split(".")
        if len(parts) > 2:  # has subdomain
            base_domain = ".".join(parts[-2:])
            if base_domain not in host_variants:
                host_variants.append(base_domain)

    for scheme in schemes:
        for host in host_variants:
            # Full path if host matches original
            if host == parsed.hostname and parsed.path not in ("", "/"):
                yield f"{scheme}://{host}{parsed.path}"
            # Root domain only
            yield f"{scheme}://{host}"

def check_with_fallbacks(url):
    """Try URL and fallbacks; return (working_url, error)."""
    last_error = None
    for candidate in generate_fallbacks(url):
        working, error = try_url(candidate)
        if working:
            return working, None
        last_error = error
    return None, last_error or "Unknown error"

def main():
    if not SEED_FILE.exists():
        print(f"Seed file not found: {SEED_FILE}")
        return

    urls = SEED_FILE.read_text().splitlines()
    working_urls = []

    with concurrent.futures.ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        futures = {executor.submit(check_with_fallbacks, u): u for u in urls}
        for future in concurrent.futures.as_completed(futures):
            original_url = futures[future]
            try:
                checked_url, error = future.result()
                if error is None:
                    working_urls.append(checked_url)
                else:
                    print(f"{original_url}: {error}")
            except Exception as e:
                print(f"{original_url}: Unexpected error {type(e).__name__}: {e}")

    WORKING_FILE.write_text("\n".join(working_urls))
    print(f"\nWorking URLs saved to {WORKING_FILE}")

if __name__ == "__main__":
    main()
