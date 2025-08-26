# Taxonomist - Developer Blog Indexer

Taxonomist is the indexing service for cozynet.dev that processes crawled developer blog content from Redis queues and stores it in PostgreSQL for full-text search.

## Architecture

The taxonomist service:
1. Consumes filtered pages from the `taxonomist_queue` Redis queue (populated by the fungicide classifier)
2. Validates and processes page content 
3. Stores structured data in PostgreSQL with full-text search capabilities
4. Provides deduplication using content and URL hashing
5. Maintains processing statistics and performance metrics

## Features

- **Full-text search** with PostgreSQL's native search capabilities
- **Content deduplication** using SHA-256 hashing
- **Batch processing** for efficient database operations  
- **Async/await architecture** for high throughput
- **Comprehensive logging** and statistics tracking
- **Graceful shutdown** handling
- **CLI tools** for database management and search testing

## Quick Start

1. **Install dependencies:**
   ```bash
   pip install -e .
   ```

2. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your PostgreSQL and Redis credentials
   ```

3. **Initialize database:**
   ```bash
   python -m src.cli init-db
   ```

4. **Run the indexer:**
   ```bash
   python main.py
   ```

## Configuration

Environment variables (see `.env.example`):

- `POSTGRES_*` - PostgreSQL connection settings
- `REDIS_*` - Redis connection and queue settings  
- `INDEXER_*` - Processing configuration (batch size, workers, timeouts)
- `MIN/MAX_CONTENT_LENGTH` - Content validation thresholds
- `LOG_LEVEL` - Logging verbosity

## Database Schema

The PostgreSQL schema includes:

- **`pages`** - Main table with full-text search vectors
- **`indexer_stats`** - Daily processing statistics
- **Indexes** - GIN indexes for search performance
- **Functions** - Automatic domain extraction and word counting
- **Triggers** - Auto-updating search vectors and metadata

## CLI Usage

```bash
# Search indexed pages
python -m src.cli search "react hooks" 10

# View statistics  
python -m src.cli stats

# Initialize database schema
python -m src.cli init-db
```

## Data Flow

```
Redis taxonomist_queue → Taxonomist Indexer → PostgreSQL pages table
                                          ↓
                               Full-text search ready
```

## Performance

- Processes ~100-1000 pages per second depending on content size
- Batch processing reduces database overhead
- Content validation prevents indexing of low-quality pages
- Async architecture enables high concurrent processing

## Monitoring

The service provides:
- Real-time processing statistics in logs
- Daily statistics stored in `indexer_stats` table
- Performance metrics (processing time per batch)
- Queue depth monitoring via Redis

## Development

The codebase structure:
- `main.py` - Entry point and service initialization
- `src/indexer.py` - Core indexing logic and Redis queue processing  
- `src/database.py` - PostgreSQL operations and connection management
- `src/models.py` - Pydantic data models for validation
- `src/config.py` - Configuration management from environment
- `src/cli.py` - Command-line utilities
- `schema.sql` - PostgreSQL database schema
