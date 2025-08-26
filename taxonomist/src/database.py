"""Database operations for the taxonomist indexer."""
import asyncio
import logging
from typing import Optional, List, Tuple
from contextlib import asynccontextmanager

import asyncpg
from asyncpg import Connection, Pool

from .config import settings
from .models import IndexedPage, IndexerStats

logger = logging.getLogger(__name__)


class DatabaseManager:
    """Manages PostgreSQL database connections and operations."""

    def __init__(self):
        self.pool: Optional[Pool] = None

    async def initialize(self) -> None:
        """Initialize the database connection pool."""
        try:
            self.pool = await asyncpg.create_pool(
                settings.postgres_url,
                min_size=2,
                max_size=settings.max_workers * 2,
                command_timeout=60,
                server_settings={
                    'application_name': 'taxonomist-indexer'
                }
            )
            logger.info("Database pool initialized successfully")

            # Test connection and run schema
            async with self.get_connection() as conn:
                await self.ensure_schema(conn)

        except Exception as e:
            logger.error(f"Failed to initialize database: {e}")
            raise

    async def close(self) -> None:
        """Close the database connection pool."""
        if self.pool:
            await self.pool.close()
            logger.info("Database pool closed")

    @asynccontextmanager
    async def get_connection(self):
        """Get a database connection from the pool."""
        if not self.pool:
            raise RuntimeError("Database pool not initialized")

        async with self.pool.acquire() as conn:
            yield conn

    async def ensure_schema(self, conn: Connection) -> None:
        """Ensure database schema is up to date."""
        try:
            # Read and execute schema file
            schema_path = "/Users/ppanda/Code/yuzhoumo/cozynet/taxonomist/schema.sql"
            with open(schema_path, 'r') as f:
                schema_sql = f.read()

            await conn.execute(schema_sql)
            logger.info("Database schema updated successfully")

        except Exception as e:
            logger.error(f"Failed to update database schema: {e}")
            raise

    async def insert_or_update_page(self, conn: Connection, page: IndexedPage) -> str:
        """
        Insert or update a page in the database.
        Returns: 'inserted', 'updated', or 'skipped'
        """
        try:
            # Check if page already exists by URL hash
            existing = await conn.fetchrow(
                "SELECT id, content_hash FROM pages WHERE url_hash = $1",
                page.url_hash
            )

            if existing:
                # Skip if content hasn't changed
                if existing['content_hash'] == page.content_hash:
                    return 'skipped'

                # Update existing page
                await conn.execute("""
                    UPDATE pages SET
                        url = $2, title = $3, description = $4, author = $5,
                        headings = $6, content = $7, keywords = $8, links = $9,
                        script_links = $10, content_hash = $11, created_at = $12,
                        updated_at = NOW()
                    WHERE url_hash = $1
                """,
                    page.url_hash, page.url, page.title, page.description,
                    page.author, page.headings, page.content, page.keywords,
                    page.links, page.script_links, page.content_hash, page.created_at
                )
                return 'updated'

            else:
                # Insert new page
                await conn.execute("""
                    INSERT INTO pages (
                        url, url_hash, title, description, author, headings,
                        content, keywords, links, script_links, content_hash, created_at
                    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
                """,
                    page.url, page.url_hash, page.title, page.description,
                    page.author, page.headings, page.content, page.keywords,
                    page.links, page.script_links, page.content_hash, page.created_at
                )
                return 'inserted'

        except Exception as e:
            logger.error(f"Failed to insert/update page {page.url}: {e}")
            raise

    async def batch_insert_or_update(
        self,
        pages: List[IndexedPage]
    ) -> Tuple[int, int, int]:
        """
        Insert or update multiple pages in a transaction.
        Returns: (inserted_count, updated_count, skipped_count)
        """
        inserted = updated = skipped = 0

        async with self.get_connection() as conn:
            async with conn.transaction():
                for page in pages:
                    result = await self.insert_or_update_page(conn, page)
                    if result == 'inserted':
                        inserted += 1
                    elif result == 'updated':
                        updated += 1
                    elif result == 'skipped':
                        skipped += 1

        return inserted, updated, skipped

    async def update_stats(
        self,
        processed: int = 0,
        inserted: int = 0,
        updated: int = 0,
        skipped: int = 0,
        errors: int = 0,
        processing_time_ms: Optional[float] = None
    ) -> None:
        """Update indexer statistics."""
        try:
            async with self.get_connection() as conn:
                await conn.execute("""
                    SELECT update_indexer_stats($1, $2, $3, $4, $5, $6)
                """, processed, inserted, updated, skipped, errors, processing_time_ms)

        except Exception as e:
            logger.error(f"Failed to update stats: {e}")

    async def get_stats(self) -> Optional[IndexerStats]:
        """Get current day's indexer statistics."""
        try:
            async with self.get_connection() as conn:
                row = await conn.fetchrow("""
                    SELECT pages_processed, pages_inserted, pages_updated,
                           pages_skipped, processing_errors, avg_processing_time_ms
                    FROM indexer_stats
                    WHERE date = CURRENT_DATE
                """)

                if row:
                    return IndexerStats(**dict(row))
                return IndexerStats()

        except Exception as e:
            logger.error(f"Failed to get stats: {e}")
            return None

    async def get_page_count(self) -> int:
        """Get total number of indexed pages."""
        try:
            async with self.get_connection() as conn:
                count = await conn.fetchval("SELECT COUNT(*) FROM pages")
                return count or 0

        except Exception as e:
            logger.error(f"Failed to get page count: {e}")
            return 0

    async def search_pages(
        self,
        query: str,
        limit: int = 20,
        offset: int = 0
    ) -> List[dict]:
        """Search pages using full-text search."""
        try:
            async with self.get_connection() as conn:
                rows = await conn.fetch("""
                    SELECT id, url, title, description, author, domain,
                           word_count, created_at,
                           ts_rank_cd(search_vector, plainto_tsquery('english', $1)) as rank
                    FROM pages
                    WHERE search_vector @@ plainto_tsquery('english', $1)
                    ORDER BY rank DESC, created_at DESC
                    LIMIT $2 OFFSET $3
                """, query, limit, offset)

                return [dict(row) for row in rows]

        except Exception as e:
            logger.error(f"Failed to search pages: {e}")
            return []


# Global database manager instance
db_manager = DatabaseManager()
