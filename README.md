Wikipedia SQLite Server
=======================

A high-performance Wikipedia dump parser and REST API server using SQLite for efficient storage and retrieval.

## Features

- **Low Memory Usage**: Efficiently processes large Wikipedia dumps with minimal memory footprint
- **SQLite Storage**: Uses SQLite with FTS5 for fast full-text search
- **REST API**: JSON-based API for searching and retrieving articles
- **Web Interface**: Built-in demo page for testing and exploration
- **Batch Processing**: Optimized batch inserts for fast indexing

## Prerequisites

- Go 1.22 or later
- SQLite 3 (with FTS5 support)
- Wikipedia dump files:
  - `articles-multistream-index.txt.bz2` (index file)
  - `articles-multistream.xml.bz2` (articles dump)

## Configuration

1. Clone this repository
2. Copy `.env.example` to `.env`
3. Set `DUMP_PATH` to the directory containing your Wikipedia dump files
4. Optionally customize `INDEX_FILE` and `ARTICLES_FILE` if using different filenames

## Usage

### Preprocessing (One-time setup)

First, load the index into the database:

```bash
go run . -load-index
```

Then, process the articles dump:

```bash
go run . -process-articles
```

You can limit the number of entries for testing:

```bash
go run . -load-index -limit 10000
go run . -process-articles -limit 1000
```

### Running the Server

After preprocessing, start the server:

```bash
go run .
```

Or build and run:

```bash
go build -o wikipedia_sqlite .
./wikipedia_sqlite
```

The server will start on port 9096. Navigate to http://localhost:9096 to access the web interface.

## API Endpoints

### Search Articles

```
GET /api/search?q=<query>&limit=<limit>
```

Search for article titles using full-text search.

**Parameters:**
- `q` (required): Search query
- `limit` (optional): Maximum number of results (default: 20)

**Example:**
```bash
curl "http://localhost:9096/api/search?q=python&limit=10"
```

**Response:**
```json
{
  "query": "python",
  "results": ["Python (programming language)", "Python", ...],
  "count": 10
}
```

### Get Article by Title

```
GET /api/article?title=<title>
```

Retrieve an article by its title.

**Parameters:**
- `title` (required): Article title

**Example:**
```bash
curl "http://localhost:9096/api/article?title=Python%20(programming%20language)"
```

**Response:**
```json
{
  "id": 12345,
  "title": "Python (programming language)",
  "namespace": 0,
  "content": "...",
  "redirect": ""
}
```

### Get Article by ID

```
GET /api/article/<id>
```

Retrieve an article by its ID.

**Example:**
```bash
curl "http://localhost:9096/api/article/12345"
```

## Docker

### Build and Run

```bash
docker-compose up --build
```

### Preprocessing in Docker

To preprocess the data in Docker:

```bash
# Load index
docker-compose run --rm wikipedia_sqlite /server -load-index

# Process articles
docker-compose run --rm wikipedia_sqlite /server -process-articles
```

## Technical Details

### Storage

- **SQLite Database**: Stores articles and index entries
- **FTS5 Virtual Table**: Enables fast full-text search
- **WAL Mode**: Write-Ahead Logging for better concurrency
- **Indexes**: Optimized indexes on title, namespace, and redirect fields

### Memory Optimization

- Batch processing with configurable batch sizes
- Streaming XML parsing (doesn't load entire dump into memory)
- SQLite cache size limits
- Efficient index lookups

### Performance

- Index loading: ~100k entries/second
- Article processing: Limited by XML parsing and disk I/O
- Search queries: Sub-second response times with FTS5
- Article retrieval: Instant with proper indexing

## License

See LICENSE file for details.


