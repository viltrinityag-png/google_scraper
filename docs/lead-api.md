# Universal Lead Search API

This repository includes an AI-agent-friendly REST API on top of the existing Google Maps scraper.

You can describe lead requests in natural language, for example:

- Find 50 dentists in Harare with websites and emails.
- Find 30 plumbers in Johannesburg.
- Find 100 restaurants in Cape Town with phone numbers.
- Find 25 car dealerships in Lusaka and return their websites.

## Run locally

### Windows

1. Install and start Docker Desktop.
2. Copy `.env.example` to `.env`.
3. Change `LEAD_API_KEY` in `.env` to a long random secret.
4. Run `START_LEAD_API.ps1`, or double-click `START_LEAD_API.bat`.

The API will be available at `http://localhost:8080`.

### Manual Docker Compose

~~~bash
cp .env.example .env
# edit .env
docker compose up --build
~~~

Results persist in the local `gmapsdata/` directory.

## API

### Health

`GET /api/v1/health`

Returns:

~~~json
{"status":"ok","service":"google-maps-lead-search"}
~~~

### Start a search

`POST /api/v1/lead-search`

Headers:

~~~text
Authorization: Bearer YOUR_LEAD_API_KEY
Content-Type: application/json
~~~

Body:

~~~json
{
  "business": "dentists",
  "location": "Harare, Zimbabwe",
  "limit": 50,
  "email": true,
  "depth": 5
}
~~~

The response is asynchronous:

~~~json
{
  "id": "JOB-ID",
  "status": "pending",
  "search": "dentists in Harare, Zimbabwe",
  "result_url": "https://YOUR-DOMAIN/api/v1/lead-search/JOB-ID?limit=50"
}
~~~

### Get results

`GET /api/v1/lead-search/{id}?limit=50`

Poll until `status` is `ok` or `failed`.

A completed response looks like:

~~~json
{
  "id": "JOB-ID",
  "status": "ok",
  "search": "dentists in Harare, Zimbabwe",
  "count": 50,
  "requested": 50,
  "results": [
    {
      "title": "Example Dental Clinic",
      "address": "...",
      "phone": "...",
      "website": "...",
      "emails": "..."
    }
  ]
}
~~~

Email extraction requires `"email": true` and can take longer.

## Production / AI-agent deployment

Do not expose `localhost` to an external agent. Deploy this repository on a server with Docker and give it a public HTTPS URL, for example:

~~~text
https://leads.example.com
~~~

Set:

~~~text
LEAD_API_KEY=use-a-long-random-secret
PUBLIC_BASE_URL=https://leads.example.com
~~~

Then run:

~~~bash
docker compose up -d --build
~~~

The API accepts either:

~~~text
Authorization: Bearer YOUR_LEAD_API_KEY
~~~

or:

~~~text
X-API-Key: YOUR_LEAD_API_KEY
~~~

## Connecting an AI agent

The OpenAPI 3 specification is `api/docs/lead-search.yaml`.

Before importing it into an API/action integration, change `servers.url` from `https://YOUR-SCRAPER-DOMAIN` to your real HTTPS URL.

The agent workflow is:

1. Interpret the requested business type and location.
2. Choose or preserve the requested lead count.
3. Set `email=true` when emails are requested.
4. Call `POST /api/v1/lead-search`.
5. Poll the returned `result_url` until the job finishes.
6. Return the requested fields in a clean format.
7. Never expose the API key in the conversation.

## Operational notes

The API does not guarantee that every business has an email or website. Results depend on information available from Google Maps and target websites. Larger searches can take longer.

For public deployment, use HTTPS, keep the API key private, monitor CPU/RAM, and configure appropriate scraping/rate limits for larger workloads.
