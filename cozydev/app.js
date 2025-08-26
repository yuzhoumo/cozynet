class CozyNetSearch {
    constructor() {
        this.apiBaseUrl = 'http://localhost:8000';
        this.currentPage = 1;
        this.perPage = 10;
        this.currentQuery = '';
        this.currentDomain = '';
        this.currentMinWords = null;
        
        this.initializeElements();
        this.bindEvents();
        this.loadStats();
    }

    initializeElements() {
        this.searchForm = document.getElementById('search-form');
        this.searchInput = document.getElementById('search-input');
        this.domainFilter = document.getElementById('domain-filter');
        this.minWordsFilter = document.getElementById('min-words');
        this.loading = document.getElementById('loading');
        this.error = document.getElementById('error');
        this.resultsContainer = document.getElementById('results-container');
        this.resultsInfo = document.getElementById('results-info');
        this.results = document.getElementById('results');
        this.pagination = document.getElementById('pagination');
        this.stats = document.getElementById('stats');
    }

    bindEvents() {
        this.searchForm.addEventListener('submit', (e) => {
            e.preventDefault();
            this.performSearch();
        });

        this.searchInput.addEventListener('input', () => {
            if (this.searchInput.value.trim() === '') {
                this.hideResults();
            }
        });
    }

    async loadStats() {
        try {
            const response = await fetch(`${this.apiBaseUrl}/stats`, {
                method: 'GET',
                mode: 'cors'
            });
            const stats = await response.json();
            
            this.stats.innerHTML = `
                <span>${stats.total_pages.toLocaleString()} pages indexed</span>
                <span>${stats.unique_domains} domains</span>
                <span>Avg ${Math.round(stats.average_word_count)} words/page</span>
            `;
        } catch (error) {
            this.stats.innerHTML = '<span>Stats unavailable</span>';
        }
    }

    async performSearch(page = 1) {
        const query = this.searchInput.value.trim();
        if (!query) return;

        this.currentQuery = query;
        this.currentPage = page;
        this.currentDomain = this.domainFilter.value.trim();
        this.currentMinWords = this.minWordsFilter.value ? parseInt(this.minWordsFilter.value) : null;

        this.showLoading();
        this.hideError();

        try {
            const params = new URLSearchParams({
                q: query,
                page: page.toString(),
                per_page: this.perPage.toString()
            });

            if (this.currentDomain) {
                params.append('domain', this.currentDomain);
            }

            if (this.currentMinWords) {
                params.append('min_words', this.currentMinWords.toString());
            }

            const response = await fetch(`${this.apiBaseUrl}/search?${params}`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                },
                mode: 'cors'
            });
            
            if (!response.ok) {
                throw new Error(`Search failed: ${response.status} ${response.statusText}`);
            }

            const data = await response.json();
            this.displayResults(data);

        } catch (error) {
            this.showError(`Search failed: ${error.message}. Make sure the API server is running on ${this.apiBaseUrl}`);
        } finally {
            this.hideLoading();
        }
    }

    displayResults(data) {
        this.resultsContainer.classList.remove('hidden');
        
        this.resultsInfo.innerHTML = `
            Found ${data.total.toLocaleString()} results 
            (showing ${((data.page - 1) * data.per_page) + 1}-${Math.min(data.page * data.per_page, data.total)})
        `;

        this.results.innerHTML = '';

        data.results.forEach(result => {
            const resultElement = this.createResultElement(result);
            this.results.appendChild(resultElement);
        });

        this.renderPagination(data);
    }

    createResultElement(result) {
        const div = document.createElement('div');
        div.className = 'result-item';
        
        const domain = result.domain || new URL(result.url).hostname;
        const title = result.title || 'Untitled';
        const description = result.description || result.content_summary;
        const createdDate = new Date(result.created_at * 1000).toLocaleDateString();

        div.innerHTML = `
            <div class="result-header">
                <h3><a href="${result.url}" target="_blank" rel="noopener">${this.escapeHtml(title)}</a></h3>
                <div class="result-meta">
                    <span class="domain">${domain}</span>
                    <span class="word-count">${result.word_count} words</span>
                    <span class="date">${createdDate}</span>
                    <span class="rank">Score: ${result.rank.toFixed(3)}</span>
                </div>
            </div>
            <p class="result-url">${result.url}</p>
            <p class="result-description">${this.escapeHtml(description)}</p>
        `;

        return div;
    }

    renderPagination(data) {
        if (data.total <= data.per_page) {
            this.pagination.innerHTML = '';
            return;
        }

        const totalPages = Math.ceil(data.total / data.per_page);
        let paginationHtml = '<div class="pagination-controls">';

        if (data.has_prev) {
            paginationHtml += `<button onclick="search.performSearch(${data.page - 1})">← Previous</button>`;
        }

        const startPage = Math.max(1, data.page - 2);
        const endPage = Math.min(totalPages, data.page + 2);

        if (startPage > 1) {
            paginationHtml += `<button onclick="search.performSearch(1)">1</button>`;
            if (startPage > 2) {
                paginationHtml += '<span>...</span>';
            }
        }

        for (let i = startPage; i <= endPage; i++) {
            const isActive = i === data.page ? ' active' : '';
            paginationHtml += `<button class="page-btn${isActive}" onclick="search.performSearch(${i})">${i}</button>`;
        }

        if (endPage < totalPages) {
            if (endPage < totalPages - 1) {
                paginationHtml += '<span>...</span>';
            }
            paginationHtml += `<button onclick="search.performSearch(${totalPages})">${totalPages}</button>`;
        }

        if (data.has_next) {
            paginationHtml += `<button onclick="search.performSearch(${data.page + 1})">Next →</button>`;
        }

        paginationHtml += '</div>';
        this.pagination.innerHTML = paginationHtml;
    }

    showLoading() {
        this.loading.classList.remove('hidden');
        this.resultsContainer.classList.add('hidden');
    }

    hideLoading() {
        this.loading.classList.add('hidden');
    }

    showError(message) {
        this.error.textContent = message;
        this.error.classList.remove('hidden');
    }

    hideError() {
        this.error.classList.add('hidden');
    }

    hideResults() {
        this.resultsContainer.classList.add('hidden');
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

const search = new CozyNetSearch();