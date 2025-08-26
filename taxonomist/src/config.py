"""Configuration management for the taxonomist indexer."""
import os
from typing import Optional
from pydantic import Field
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    # Redis Configuration
    redis_host: str = Field(default="localhost", env="REDIS_HOST")
    redis_port: int = Field(default=6379, env="REDIS_PORT")
    redis_password: Optional[str] = Field(default=None, env="REDIS_PASSWORD")
    redis_db: int = Field(default=0, env="REDIS_DB")
    redis_taxonomist_queue_key: str = Field(
        default="taxonomist_queue",
        env="REDIS_TAXONOMIST_QUEUE_KEY"
    )

    # PostgreSQL Configuration
    postgres_host: str = Field(default="localhost", env="POSTGRES_HOST")
    postgres_port: int = Field(default=5432, env="POSTGRES_PORT")
    postgres_user: str = Field(default="postgres", env="POSTGRES_USER")
    postgres_password: str = Field(..., env="POSTGRES_PASSWORD")
    postgres_database: str = Field(default="cozynet", env="POSTGRES_DATABASE")

    # Indexer Configuration
    batch_size: int = Field(default=10, env="INDEXER_BATCH_SIZE")
    max_workers: int = Field(default=4, env="INDEXER_MAX_WORKERS")
    queue_timeout: int = Field(default=30, env="INDEXER_QUEUE_TIMEOUT")
    min_content_length: int = Field(default=100, env="MIN_CONTENT_LENGTH")
    max_content_length: int = Field(default=50000, env="MAX_CONTENT_LENGTH")

    # Logging
    log_level: str = Field(default="INFO", env="LOG_LEVEL")

    @property
    def redis_url(self) -> str:
        """Get Redis connection URL."""
        auth = f":{self.redis_password}@" if self.redis_password else ""
        return f"redis://{auth}{self.redis_host}:{self.redis_port}/{self.redis_db}"

    @property
    def postgres_url(self) -> str:
        """Get PostgreSQL connection URL."""
        return (
            f"postgresql://{self.postgres_user}:{self.postgres_password}"
            f"@{self.postgres_host}:{self.postgres_port}/{self.postgres_database}"
        )

    class Config:
        env_file = ".env"
        case_sensitive = False
        extra = "ignore"


# Global settings instance
settings = Settings()
