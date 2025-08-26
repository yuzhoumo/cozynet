# cozynet.dev

> "The original idea of the web was that it should be a collaborative space
> where you can communicate through sharing information." - Tim Berners-Lee

Cozynet.dev is a search engine designed to index and surface personal
developer blogs. The public web has grown into a cesspool dominated by
advertisements, spam, marketers, bots, and trolls. Top search engine results
are saturated with SEO-optimized listicles, AI-generated slop, and paid
promotions.

Cozynet attempts to exclude sites that contain paywalls, are associated
with commercial entities, contain advertising, or track their visitors. Its
purpose is to make accessible the treasure trove of independent blogs tended
to by passionate developers seeking to share their knowledge with others free
of profit-driven intent.

## Services

- mycelium: web crawling service
- fungicide: page filtering service for the mycelium crawler queue
- taxonomist: page indexing service
- greenhouse: restful querying service
- cozynet: static frontend search page

## Architecture

![architecture diagram](./docs/diagram.svg)
