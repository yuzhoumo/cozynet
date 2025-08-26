"""Core indexer service for processing the taxonomist Redis queue."""
import asyncio
import json
import logging
import time
from typing import List, Optional
from contextlib import asynccontextmanager

import redis.asyncio as redis
from pydantic import ValidationError

from .config import settings
from .models import Page, IndexedPage
from .database import db_manager

logger = logging.getLogger(__name__)


class IndexerService:
    """Main indexer service that processes pages from Redis queue."""

    def __init__(self):
        self.redis_client: Optional[redis.Redis] = None
        self.running = False
        self._stats = {
            'processed': 0,
            'inserted': 0,
            'updated': 0,
            'skipped': 0,
            'errors': 0
        }

    async def initialize(self) -> None:
        """Initialize Redis connection and database."""
        try:
            # Initialize Redis client
            self.redis_client = redis.from_url(settings.redis_url)
            await self.redis_client.ping()
            logger.info("Redis connection established")

            # Initialize database
            await db_manager.initialize()
            logger.info("Indexer service initialized successfully")

        except Exception as e:
            logger.error(f"Failed to initialize indexer service: {e}")
            raise

    async def close(self) -> None:
        """Close connections and cleanup."""
        self.running = False

        if self.redis_client:
            await self.redis_client.close()
            logger.info("Redis connection closed")

        await db_manager.close()
        logger.info("Indexer service closed")

    @asynccontextmanager
    async def get_redis_client(self):
        """Get Redis client with connection handling."""
        if not self.redis_client:
            raise RuntimeError("Redis client not initialized")
        yield self.redis_client

    def is_valid_page(self, page: Page) -> bool:
        """Validate if a page should be indexed."""
        content_text = page.get_content_text()
        content_length = len(content_text)

        # Check content length constraints
        if content_length < settings.min_content_length:
            logger.debug(f"Skipping page {page.location}: content too short ({content_length} chars)")
            return False

        if content_length > settings.max_content_length:
            logger.debug(f"Skipping page {page.location}: content too long ({content_length} chars)")
            return False

        # Check for required fields
        if not page.title and not content_text.strip():
            logger.debug(f"Skipping page {page.location}: no title or content")
            return False

        return True

    async def process_page(self, page_data: dict) -> str:
        """
        Process a single page from the queue.
        Returns: 'inserted', 'updated', 'skipped', or 'error'
        """
        try:
            # Parse and validate page data
            page = Page(**page_data)

            # Validate page content
            if not self.is_valid_page(page):
                return 'skipped'

            # Convert to indexed page model
            indexed_page = IndexedPage.from_page(page)

            # Insert or update in database
            async with db_manager.get_connection() as conn:
                result = await db_manager.insert_or_update_page(conn, indexed_page)
                logger.debug(f"Page {page.location} - {result}")
                return result

        except ValidationError as e:
            logger.error(f"Invalid page data: {e}")
            return 'error'
        except Exception as e:
            logger.error(f"Failed to process page: {e}")
            return 'error'

    async def process_batch(self, batch: List[dict]) -> None:
        """Process a batch of pages."""
        start_time = time.time()

        try:
            # Process pages concurrently
            tasks = [self.process_page(page_data) for page_data in batch]
            results = await asyncio.gather(*tasks, return_exceptions=True)

            # Count results
            for result in results:
                if isinstance(result, Exception):
                    self._stats['errors'] += 1
                    logger.error(f"Batch processing error: {result}")
                else:
                    self._stats['processed'] += 1
                    if result == 'inserted':
                        self._stats['inserted'] += 1
                    elif result == 'updated':
                        self._stats['updated'] += 1
                    elif result == 'skipped':
                        self._stats['skipped'] += 1
                    elif result == 'error':
                        self._stats['errors'] += 1

            # Calculate processing time
            processing_time = (time.time() - start_time) * 1000

            # Update statistics
            await db_manager.update_stats(
                processed=len([r for r in results if not isinstance(r, Exception)]),
                inserted=len([r for r in results if r == 'inserted']),
                updated=len([r for r in results if r == 'updated']),
                skipped=len([r for r in results if r == 'skipped']),
                errors=len([r for r in results if isinstance(r, Exception) or r == 'error']),
                processing_time_ms=processing_time
            )

            logger.info(f"Processed batch of {len(batch)} pages in {processing_time:.2f}ms")

        except Exception as e:
            logger.error(f"Batch processing failed: {e}")
            self._stats['errors'] += len(batch)

    async def get_queue_item(self) -> Optional[dict]:
        """Get a single item from the Redis queue."""
        try:
            async with self.get_redis_client() as redis_client:
                # Use blocking pop with timeout
                result = await redis_client.blpop(
                    settings.redis_taxonomist_queue_key,
                    timeout=settings.queue_timeout
                )

                if result:
                    queue_name, item_data = result
                    return json.loads(item_data)
                return None

        except json.JSONDecodeError as e:
            logger.error(f"Invalid JSON in queue: {e}")
            return None
        except Exception as e:
            logger.error(f"Failed to get queue item: {e}")
            return None

    async def get_batch(self) -> List[dict]:
        """Get a batch of items from the Redis queue."""
        batch = []

        # Get first item (blocking)
        first_item = await self.get_queue_item()
        if first_item:
            batch.append(first_item)

        # Get additional items for batch (non-blocking)
        for _ in range(settings.batch_size - 1):
            try:
                async with self.get_redis_client() as redis_client:
                    result = await redis_client.lpop(settings.redis_taxonomist_queue_key)
                    if result:
                        item_data = json.loads(result)
                        batch.append(item_data)
                    else:
                        break
            except (json.JSONDecodeError, Exception) as e:
                logger.error(f"Error getting batch item: {e}")
                break

        return batch

    async def log_stats(self) -> None:
        """Log current processing statistics."""
        total_pages = await db_manager.get_page_count()
        stats = await db_manager.get_stats()

        logger.info(
            f"Session: {self._stats['processed']} processed, "
            f"{self._stats['inserted']} inserted, "
            f"{self._stats['updated']} updated, "
            f"{self._stats['skipped']} skipped, "
            f"{self._stats['errors']} errors"
        )

        if stats:
            logger.info(
                f"Today: {stats.pages_processed} processed, "
                f"{stats.pages_inserted} inserted, "
                f"{stats.pages_updated} updated, "
                f"{stats.pages_skipped} skipped, "
                f"{stats.processing_errors} errors"
            )

        logger.info(f"Total pages in database: {total_pages}")

    async def run(self) -> None:
        """Main indexer loop."""
        self.running = True
        logger.info("Starting indexer service...")

        # Log initial statistics
        await self.log_stats()

        stats_log_interval = 100  # Log stats every 100 processed items
        processed_since_log = 0

        try:
            while self.running:
                # Get batch of pages from queue
                batch = await self.get_batch()

                if not batch:
                    logger.debug("No items in queue, waiting...")
                    await asyncio.sleep(1)
                    continue

                # Process batch
                await self.process_batch(batch)
                processed_since_log += len(batch)

                # Log periodic statistics
                if processed_since_log >= stats_log_interval:
                    await self.log_stats()
                    processed_since_log = 0

        except KeyboardInterrupt:
            logger.info("Received shutdown signal")
        except Exception as e:
            logger.error(f"Indexer error: {e}")
        finally:
            logger.info("Indexer service stopped")
            await self.log_stats()


# Global indexer service instance
indexer_service = IndexerService()
