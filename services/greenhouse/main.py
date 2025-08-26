import os
import hashlib
from datetime import datetime
from typing import List, Optional, Dict, Any
from contextlib import asynccontextmanager

import asyncpg
from fastapi import FastAPI, Query, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

# Load environment variables from .env file
try:
    from dotenv import load_dotenv
    load_dotenv()
except ImportError:
    pass


class SearchResult(BaseModel):
    id: str
    url: str
    title: Optional[str]
    description: Optional[str]
    content_summary: str
    domain: str
    word_count: int
    created_at: int
    rank: float


class SearchResponse(BaseModel):
    results: List[SearchResult]
    total: int
    page: int
    per_page: int
    has_next: bool
    has_prev: bool


@asynccontextmanager
async def lifespan(app: FastAPI):
    try:
        app.state.db_pool = await asyncpg.create_pool(
            host=os.getenv("POSTGRES_HOST", "localhost"),
            port=int(os.getenv("POSTGRES_PORT", 5432)),
            user=os.getenv("POSTGRES_USER", "taxonomist"),
            password=os.getenv("POSTGRES_PASSWORD", "taxonomist_pass"),
            database=os.getenv("POSTGRES_DATABASE", "taxonomist"),
            min_size=5,
            max_size=20
        )
    except Exception as e:
        print(f"Warning: Could not connect to database: {e}")
        print("API will run in demo mode without database connection")
        app.state.db_pool = None
    yield
    if app.state.db_pool:
        await app.state.db_pool.close()


app = FastAPI(
    title="Greenhouse Search API",
    description="REST API for querying the CozyNet search engine",
    version="1.0.0",
    lifespan=lifespan
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


def create_content_summary(content: str, max_length: int = 200) -> str:
    """Create a summary of the content, similar to Google search snippets."""
    if not content:
        return ""

    content = content.strip()
    if len(content) <= max_length:
        return content

    words = content.split()
    summary = ""
    for word in words:
        if len(summary) + len(word) + 1 > max_length:
            break
        summary += word + " "

    return summary.strip() + "..."


@app.get("/search", response_model=SearchResponse)
async def search(
    q: str = Query(..., description="Search query"),
    page: int = Query(1, ge=1, description="Page number (1-based)"),
    per_page: int = Query(10, ge=1, le=100, description="Results per page"),
    domain: Optional[str] = Query(None, description="Filter by domain"),
    min_words: Optional[int] = Query(None, description="Minimum word count")
):
    """Search for pages using full-text search with pagination."""
    if not app.state.db_pool:
        # Demo mode - return sample data
        sample_results = [
            SearchResult(
                id="demo-1",
                url="https://example.com/demo-page",
                title="Demo Search Result",
                description="This is a demo search result for testing the API",
                content_summary="This is sample content that would normally come from the search index. It demonstrates how the API structures search results with pagination...",
                domain="example.com",
                word_count=150,
                created_at=1640995200,
                rank=0.95
            )
        ]

        return SearchResponse(
            results=sample_results,
            total=1,
            page=page,
            per_page=per_page,
            has_next=False,
            has_prev=False
        )

    offset = (page - 1) * per_page

    async with app.state.db_pool.acquire() as conn:
        base_query = """
        SELECT
            id::text,
            url,
            title,
            description,
            content,
            domain,
            word_count,
            created_at,
            ts_rank_cd(search_vector, plainto_tsquery('english', $1)) as rank
        FROM pages
        WHERE search_vector @@ plainto_tsquery('english', $1)
        """

        count_query = """
        SELECT COUNT(*)
        FROM pages
        WHERE search_vector @@ plainto_tsquery('english', $1)
        """

        params = [q]
        param_idx = 2

        if domain:
            base_query += f" AND domain = ${param_idx}"
            count_query += f" AND domain = ${param_idx}"
            params.append(domain)
            param_idx += 1

        if min_words:
            base_query += f" AND word_count >= ${param_idx}"
            count_query += f" AND word_count >= ${param_idx}"
            params.append(min_words)
            param_idx += 1

        base_query += f" ORDER BY rank DESC LIMIT ${param_idx} OFFSET ${param_idx + 1}"
        params.extend([per_page, offset])

        try:
            total = await conn.fetchval(count_query, *params[:-2])
            rows = await conn.fetch(base_query, *params)

            results = [
                SearchResult(
                    id=row['id'],
                    url=row['url'],
                    title=row['title'],
                    description=row['description'],
                    content_summary=create_content_summary(row['content']),
                    domain=row['domain'],
                    word_count=row['word_count'],
                    created_at=row['created_at'],
                    rank=float(row['rank'])
                )
                for row in rows
            ]

            return SearchResponse(
                results=results,
                total=total,
                page=page,
                per_page=per_page,
                has_next=offset + per_page < total,
                has_prev=page > 1
            )

        except Exception as e:
            raise HTTPException(status_code=500, detail=f"Database error: {str(e)}")


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    if not app.state.db_pool:
        return {"status": "demo_mode", "timestamp": datetime.utcnow().isoformat()}

    try:
        async with app.state.db_pool.acquire() as conn:
            await conn.fetchval("SELECT 1")
        return {"status": "healthy", "timestamp": datetime.utcnow().isoformat()}
    except Exception as e:
        raise HTTPException(status_code=503, detail=f"Database connection failed: {str(e)}")


@app.get("/stats")
async def get_stats():
    """Get search index statistics."""
    if not app.state.db_pool:
        return {
            "total_pages": 1,
            "average_word_count": 150.0,
            "unique_domains": 1,
            "latest_page_timestamp": 1640995200,
            "mode": "demo"
        }

    async with app.state.db_pool.acquire() as conn:
        try:
            total_pages = await conn.fetchval("SELECT COUNT(*) FROM pages")
            avg_word_count = await conn.fetchval("SELECT AVG(word_count) FROM pages")
            domains_count = await conn.fetchval("SELECT COUNT(DISTINCT domain) FROM pages")
            latest_page = await conn.fetchval("SELECT MAX(created_at) FROM pages")

            return {
                "total_pages": total_pages,
                "average_word_count": round(float(avg_word_count or 0), 2),
                "unique_domains": domains_count,
                "latest_page_timestamp": latest_page
            }
        except Exception as e:
            raise HTTPException(status_code=500, detail=f"Database error: {str(e)}")


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
