"""Taxonomist - Developer blog indexer for cozynet.dev"""
import asyncio
import logging
import signal
import sys
from pathlib import Path

# Add src directory to Python path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from src.config import settings
from src.indexer import indexer_service
from dotenv import load_dotenv


def setup_logging():
    """Configure logging for the application."""
    logging.basicConfig(
        level=getattr(logging, settings.log_level.upper()),
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=[
            logging.StreamHandler(),
            logging.FileHandler('taxonomist.log')
        ]
    )
    
    # Reduce noise from external libraries
    logging.getLogger('asyncpg').setLevel(logging.WARNING)
    logging.getLogger('redis').setLevel(logging.WARNING)


async def shutdown_handler(service):
    """Handle graceful shutdown."""
    logging.info("Shutting down gracefully...")
    await service.close()
    sys.exit(0)


async def main():
    """Main entry point for the taxonomist indexer."""
    # Load environment variables
    load_dotenv()
    
    # Setup logging
    setup_logging()
    logger = logging.getLogger(__name__)
    
    logger.info("Starting Taxonomist - Developer Blog Indexer")
    logger.info(f"Config: batch_size={settings.batch_size}, max_workers={settings.max_workers}")
    
    try:
        # Initialize the indexer service
        await indexer_service.initialize()
        
        # Setup graceful shutdown handlers
        for sig in [signal.SIGTERM, signal.SIGINT]:
            signal.signal(sig, lambda s, f: asyncio.create_task(shutdown_handler(indexer_service)))
        
        # Run the indexer
        await indexer_service.run()
        
    except KeyboardInterrupt:
        logger.info("Received keyboard interrupt")
    except Exception as e:
        logger.error(f"Fatal error: {e}")
        sys.exit(1)
    finally:
        await indexer_service.close()


if __name__ == "__main__":
    asyncio.run(main())
