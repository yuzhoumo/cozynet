"""CLI utilities for the taxonomist indexer."""
import asyncio
import json
import sys
from typing import Optional

from .database import db_manager
from .config import settings


async def search_command(query: str, limit: int = 20) -> None:
    """Search indexed pages."""
    await db_manager.initialize()

    try:
        results = await db_manager.search_pages(query, limit=limit)

        if not results:
            print(f"No results found for: {query}")
            return

        print(f"Found {len(results)} results for: {query}")
        print("=" * 60)

        for i, result in enumerate(results, 1):
            print(f"{i}. {result['title'] or 'No title'}")
            print(f"   URL: {result['url']}")
            print(f"   Domain: {result['domain']}")
            print(f"   Author: {result['author'] or 'Unknown'}")
            print(f"   Words: {result['word_count']}")
            print(f"   Rank: {result['rank']:.4f}")
            if result['description']:
                desc = result['description'][:100] + "..." if len(result['description']) > 100 else result['description']
                print(f"   Description: {desc}")
            print()

    finally:
        await db_manager.close()


async def stats_command() -> None:
    """Show indexer statistics."""
    await db_manager.initialize()

    try:
        total_pages = await db_manager.get_page_count()
        stats = await db_manager.get_stats()

        print("Taxonomist Indexer Statistics")
        print("=" * 40)
        print(f"Total pages indexed: {total_pages}")

        if stats:
            print(f"\nToday's statistics:")
            print(f"  Pages processed: {stats.pages_processed}")
            print(f"  Pages inserted: {stats.pages_inserted}")
            print(f"  Pages updated: {stats.pages_updated}")
            print(f"  Pages skipped: {stats.pages_skipped}")
            print(f"  Processing errors: {stats.processing_errors}")
            if stats.avg_processing_time_ms:
                print(f"  Avg processing time: {stats.avg_processing_time_ms:.2f}ms")
        else:
            print("No statistics available for today")

    finally:
        await db_manager.close()


async def init_db_command() -> None:
    """Initialize the database schema."""
    print("Initializing database schema...")

    try:
        await db_manager.initialize()
        print("Database schema initialized successfully!")
    except Exception as e:
        print(f"Failed to initialize database: {e}")
        sys.exit(1)
    finally:
        await db_manager.close()


def main():
    """CLI entry point."""
    if len(sys.argv) < 2:
        print("Usage: python -m src.cli <command> [args]")
        print("Commands:")
        print("  search <query> [limit]  - Search indexed pages")
        print("  stats                   - Show indexer statistics")
        print("  init-db                 - Initialize database schema")
        sys.exit(1)

    command = sys.argv[1]

    if command == "search":
        if len(sys.argv) < 3:
            print("Usage: python -m src.cli search <query> [limit]")
            sys.exit(1)

        query = sys.argv[2]
        limit = int(sys.argv[3]) if len(sys.argv) > 3 else 20
        asyncio.run(search_command(query, limit))

    elif command == "stats":
        asyncio.run(stats_command())

    elif command == "init-db":
        asyncio.run(init_db_command())

    else:
        print(f"Unknown command: {command}")
        sys.exit(1)


if __name__ == "__main__":
    main()
