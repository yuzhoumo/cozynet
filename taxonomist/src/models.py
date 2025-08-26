"""Data models for the taxonomist indexer."""
import hashlib
from typing import List, Optional
from pydantic import BaseModel, Field, validator


class Page(BaseModel):
    """Model for a crawled page from the taxonomist_queue."""
    title: Optional[str] = None
    description: Optional[str] = None
    author: Optional[str] = None
    keywords: List[str] = Field(default_factory=list)
    headings: List[str] = Field(default_factory=list)
    content: List[str] = Field(default_factory=list)
    links: List[str] = Field(default_factory=list)
    script_links: List[str] = Field(default_factory=list)
    script_content: List[str] = Field(default_factory=list)
    location: str
    created_at: Optional[int] = None

    @validator('content', pre=True)
    def ensure_content_list(cls, v):
        """Ensure content is a list."""
        if isinstance(v, str):
            return [v]
        return v or []

    @validator('keywords', 'headings', 'links', 'script_links', 'script_content', pre=True)
    def ensure_list_fields(cls, v):
        """Ensure list fields are always lists, never None."""
        return v or []

    @validator('location')
    def validate_url(cls, v):
        """Basic URL validation."""
        if not v or not v.startswith(('http://', 'https://')):
            raise ValueError('Invalid URL format')
        return v

    def get_url_hash(self) -> str:
        """Generate SHA-256 hash of the URL."""
        return hashlib.sha256(self.location.encode()).hexdigest()

    def get_content_hash(self) -> str:
        """Generate SHA-256 hash of the normalized content."""
        content_text = ' '.join(self.content).lower().strip()
        title_text = (self.title or '').lower().strip()
        combined = f"{title_text} {content_text}"
        return hashlib.sha256(combined.encode()).hexdigest()

    def get_content_text(self) -> str:
        """Get concatenated content as a single string."""
        return ' '.join(self.content)

    def extract_domain(self) -> str:
        """Extract domain from URL."""
        import re
        domain = re.sub(r'^https?://', '', self.location)
        domain = re.sub(r'/.*$', '', domain)
        return domain


class IndexedPage(BaseModel):
    """Model for a page stored in PostgreSQL."""
    id: Optional[str] = None
    url: str
    url_hash: str
    title: Optional[str] = None
    description: Optional[str] = None
    author: Optional[str] = None
    headings: List[str] = Field(default_factory=list)
    content: str
    keywords: List[str] = Field(default_factory=list)
    links: List[str] = Field(default_factory=list)
    script_links: List[str] = Field(default_factory=list)
    domain: str
    content_hash: str
    word_count: int = 0
    created_at: Optional[int] = None

    @classmethod
    def from_page(cls, page: Page) -> 'IndexedPage':
        """Create IndexedPage from crawled Page."""
        import time
        return cls(
            url=page.location,
            url_hash=page.get_url_hash(),
            title=page.title,
            description=page.description,
            author=page.author,
            headings=page.headings,
            content=page.get_content_text(),
            keywords=page.keywords,
            links=page.links,
            script_links=page.script_links,
            domain=page.extract_domain(),
            content_hash=page.get_content_hash(),
            created_at=page.created_at or int(time.time())
        )


class IndexerStats(BaseModel):
    """Model for indexer processing statistics."""
    pages_processed: int = 0
    pages_inserted: int = 0
    pages_updated: int = 0
    pages_skipped: int = 0
    processing_errors: int = 0
    avg_processing_time_ms: Optional[float] = None
