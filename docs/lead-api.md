# Universal Lead Search API

This repo now includes a simple AI-agent-facing facade over the existing Google Maps scraper.

## Start it

```bash
mkdir -p gmapsdata
docker run --rm -p 8080:8080 \
  -v "$PWD/gmapsdata:/gmapsdata" \
  -e LEAD_API_KEY="replace-with-a-long-random-secret" \
  gosom/google-maps-scraper \
  -data-folder /gmapsdata -addr :8080
```

The original scraper and web UI continue to work. The new endpoint is:

- `POST /api/v1/lead-search`
- `GET /api/v1/lead-search/{id}`

Example request:

```json
{
  "business": "dentists",
  "location": "Harare, Zimbabwe",
  "limit": 50,
  "email": true,
  "depth": 5
}
```

The POST returns a job ID immediately. Poll the returned `result_url` until `status` is `ok`. The result contains the scraper's structured CSV fields, including business name, address, phone, website, rating, review count and emails when available.

Set `LEAD_API_KEY` whenever the server is reachable outside your private machine. Use `Authorization: Bearer <key>` or `X-API-Key: <key>`.

The OpenAPI schema for connecting an AI agent is `api/docs/lead-search.yaml`. Replace `YOUR-SCRAPER-DOMAIN` with the public HTTPS base URL before importing it into an integration/action.
