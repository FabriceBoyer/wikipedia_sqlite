package wikipedia

import (
	"bufio"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/d4l3k/go-pbzip2"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Wiki struct {
	dbPath       string
	db           *sql.DB
	indexFile    string
	articlesFile string
	mu           sync.RWMutex
	initialized  bool
	ftsVersion   string // "fts5", "fts4", or "none"
}

type Article struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Namespace int    `json:"namespace"`
	Content   string `json:"content"`
	Redirect  string `json:"redirect,omitempty"`
}

type IndexEntry struct {
	Seek int64
	ID   int64
}

// NewWiki creates a new Wiki instance
func NewWiki(dumpPath, indexFile, articlesFile string) *Wiki {
	return &Wiki{
		indexFile:    filepath.Join(dumpPath, indexFile),
		articlesFile: filepath.Join(dumpPath, articlesFile),
		dbPath:       filepath.Join(dumpPath, "wikipedia.db"),
	}
}

// Open initializes the database connection
func (w *Wiki) Open() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.initialized {
		return nil
	}

	var err error
	w.db, err = sql.Open("sqlite3", w.dbPath+"?_journal_mode=WAL&_sync=OFF&_cache_size=10000")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	_, err = w.db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// Set cache size to reduce memory usage
	_, err = w.db.Exec("PRAGMA cache_size=-10000") // 10MB cache
	if err != nil {
		return fmt.Errorf("failed to set cache size: %w", err)
	}

	// Create tables
	if err := w.createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	w.initialized = true
	return nil
}

// Close closes the database connection
func (w *Wiki) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db != nil {
		return w.db.Close()
	}
	return nil
}

// createTables creates the necessary database tables
func (w *Wiki) createTables() error {
	// Articles table
	createArticles := `
	CREATE TABLE IF NOT EXISTS articles (
		id INTEGER PRIMARY KEY,
		title TEXT NOT NULL,
		namespace INTEGER NOT NULL,
		content TEXT,
		redirect TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := w.db.Exec(createArticles); err != nil {
		return fmt.Errorf("failed to create articles table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_articles_title ON articles(title)",
		"CREATE INDEX IF NOT EXISTS idx_articles_namespace ON articles(namespace)",
		"CREATE INDEX IF NOT EXISTS idx_articles_redirect ON articles(redirect)",
	}

	for _, idx := range indexes {
		if _, err := w.db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Check if FTS table already exists and detect version
	var existingSQL string
	err := w.db.QueryRow(`
		SELECT sql FROM sqlite_master
		WHERE name = 'articles_fts' AND type = 'table'
	`).Scan(&existingSQL)

	ftsVersion := "none"
	if err == nil && existingSQL != "" {
		// Table exists, detect version from SQL
		if strings.Contains(strings.ToLower(existingSQL), "fts5") {
			ftsVersion = "fts5"
		} else if strings.Contains(strings.ToLower(existingSQL), "fts4") {
			ftsVersion = "fts4"
		}
		log.Printf("Detected existing %s table", ftsVersion)
	} else {
		// Table doesn't exist, try to create FTS5 first
		createFTS5 := `
		CREATE VIRTUAL TABLE IF NOT EXISTS articles_fts USING fts5(
			title,
			content,
			content_rowid=id,
			content='articles'
		)`

		if _, err := w.db.Exec(createFTS5); err != nil {
			// FTS5 not available, try FTS4
			log.Printf("FTS5 not available, trying FTS4: %v", err)
			createFTS4 := `
			CREATE VIRTUAL TABLE IF NOT EXISTS articles_fts USING fts4(
				title,
				content,
				content='articles'
			)`
			if _, err := w.db.Exec(createFTS4); err != nil {
				log.Printf("FTS4 also not available, will use LIKE queries: %v", err)
				ftsVersion = "none"
			} else {
				ftsVersion = "fts4"
				// FTS4 triggers (slightly different syntax)
				triggers := []string{
					`CREATE TRIGGER IF NOT EXISTS articles_ai AFTER INSERT ON articles BEGIN
						INSERT INTO articles_fts(docid, title, content) VALUES (new.id, new.title, new.content);
					END`,
					`CREATE TRIGGER IF NOT EXISTS articles_ad AFTER DELETE ON articles BEGIN
						DELETE FROM articles_fts WHERE docid = old.id;
					END`,
					`CREATE TRIGGER IF NOT EXISTS articles_au AFTER UPDATE ON articles BEGIN
						DELETE FROM articles_fts WHERE docid = old.id;
						INSERT INTO articles_fts(docid, title, content) VALUES (new.id, new.title, new.content);
					END`,
				}
				for _, trigger := range triggers {
					if _, err := w.db.Exec(trigger); err != nil {
						log.Printf("Warning: failed to create FTS4 trigger: %v", err)
					}
				}
			}
		} else {
			ftsVersion = "fts5"
			// FTS5 triggers
			triggers := []string{
				`CREATE TRIGGER IF NOT EXISTS articles_ai AFTER INSERT ON articles BEGIN
					INSERT INTO articles_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
				END`,
				`CREATE TRIGGER IF NOT EXISTS articles_ad AFTER DELETE ON articles BEGIN
					DELETE FROM articles_fts WHERE rowid = old.id;
				END`,
				`CREATE TRIGGER IF NOT EXISTS articles_au AFTER UPDATE ON articles BEGIN
					DELETE FROM articles_fts WHERE rowid = old.id;
					INSERT INTO articles_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
				END`,
			}
			for _, trigger := range triggers {
				if _, err := w.db.Exec(trigger); err != nil {
					log.Printf("Warning: failed to create FTS5 trigger: %v", err)
				}
			}
		}
	}

	// Store FTS version for later use
	w.ftsVersion = ftsVersion

	if w.ftsVersion != "none" {
		log.Printf("Using %s for full-text search", w.ftsVersion)
	} else {
		log.Printf("FTS not available, using LIKE-based search")
	}

	// Index entries table for fast lookup
	createIndexTable := `
	CREATE TABLE IF NOT EXISTS index_entries (
		seek INTEGER NOT NULL,
		article_id INTEGER NOT NULL,
		PRIMARY KEY (seek, article_id)
	)`

	if _, err := w.db.Exec(createIndexTable); err != nil {
		return fmt.Errorf("failed to create index_entries table: %w", err)
	}

	if _, err := w.db.Exec("CREATE INDEX IF NOT EXISTS idx_index_entries_seek ON index_entries(seek)"); err != nil {
		return fmt.Errorf("failed to create index_entries index: %w", err)
	}

	return nil
}

// LoadIndex loads the index file into the database
func (w *Wiki) LoadIndex(limit int) error {
	if err := w.Open(); err != nil {
		return err
	}

	f, err := os.Open(w.indexFile)
	if err != nil {
		return fmt.Errorf("failed to open index file: %w", err)
	}
	defer f.Close()

	r, err := pbzip2.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create bzip2 reader: %w", err)
	}
	defer r.Close()

	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 16*1024)
	scanner.Buffer(buf, 16*1024)

	log.Printf("Reading index file %s...", w.indexFile)

	// Begin transaction for batch inserts
	tx, err := w.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO index_entries (seek, article_id) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	i := 0
	batchSize := 10000

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		seek, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}

		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}

		_, err = stmt.Exec(seek, id)
		if err != nil {
			log.Printf("Error inserting index entry: %v", err)
			continue
		}

		i++
		if i%batchSize == 0 {
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit transaction: %w", err)
			}
			log.Printf("Processed %d index entries", i)

			// Start new transaction
			tx, err = w.db.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin transaction: %w", err)
			}
			stmt, err = tx.Prepare("INSERT OR IGNORE INTO index_entries (seek, article_id) VALUES (?, ?)")
			if err != nil {
				return fmt.Errorf("failed to prepare statement: %w", err)
			}
		}

		if limit > 0 && i >= limit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit final transaction: %w", err)
	}

	log.Printf("Done loading index! Processed %d entries", i)
	return nil
}

// ProcessArticles processes the articles dump and stores them in the database
// This processes the entire XML stream and stores articles that are in the index
func (w *Wiki) ProcessArticles(limit int) error {
	if err := w.Open(); err != nil {
		return err
	}

	// Get all article IDs from index to know which articles to process
	rows, err := w.db.Query("SELECT DISTINCT article_id FROM index_entries")
	if err != nil {
		return fmt.Errorf("failed to query index entries: %w", err)
	}
	defer rows.Close()

	indexSet := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		indexSet[id] = true
	}

	log.Printf("Found %d articles to process from index", len(indexSet))

	f, err := os.Open(w.articlesFile)
	if err != nil {
		return fmt.Errorf("failed to open articles file: %w", err)
	}
	defer f.Close()

	// Process articles in batches
	batchSize := 1000
	tx, err := w.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO articles (id, title, namespace, content, redirect)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Use pbzip2 for parallel decompression
	r, err := pbzip2.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create bzip2 reader: %w", err)
	}
	defer r.Close()

	decoder := xml.NewDecoder(r)
	count := 0
	processed := 0

	log.Printf("Processing articles from %s...", w.articlesFile)

	for {
		var page Page
		err := decoder.Decode(&page)
		if err != nil {
			// EOF or other error - stop processing
			if err == io.EOF {
				break
			}
			// Log non-EOF errors but continue
			log.Printf("XML decode error: %v", err)
			continue
		}

		// Only process articles in the main namespace (0)
		if page.NS != 0 {
			continue
		}

		// Check if this article is in our index
		if !indexSet[int64(page.ID)] {
			continue
		}

		redirect := ""
		if len(page.Redirect) > 0 {
			redirect = page.Redirect[0].Title
		}

		// Truncate content if too large (to avoid memory issues)
		content := page.Text
		if len(content) > 10*1024*1024 { // 10MB max per article
			content = content[:10*1024*1024]
		}

		_, err = stmt.Exec(page.ID, page.Title, page.NS, content, redirect)
		if err != nil {
			log.Printf("Error inserting article %d: %v", page.ID, err)
			continue
		}

		count++
		processed++

		if count >= batchSize {
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit transaction: %w", err)
			}
			log.Printf("Processed %d articles", processed)

			tx, err = w.db.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin transaction: %w", err)
			}
			stmt, err = tx.Prepare(`
				INSERT OR REPLACE INTO articles (id, title, namespace, content, redirect)
				VALUES (?, ?, ?, ?, ?)
			`)
			if err != nil {
				return fmt.Errorf("failed to prepare statement: %w", err)
			}
			count = 0
		}

		if limit > 0 && processed >= limit {
			break
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit final transaction: %w", err)
	}

	log.Printf("Done processing articles! Processed %d articles", processed)
	return nil
}

// GetArticle retrieves an article by title
func (w *Wiki) GetArticle(title string) (*Article, error) {
	if err := w.Open(); err != nil {
		return nil, err
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	// Try exact match first
	var article Article
	err := w.db.QueryRow(`
		SELECT id, title, namespace, content, redirect
		FROM articles
		WHERE title = ?
		LIMIT 1
	`, title).Scan(&article.ID, &article.Title, &article.Namespace, &article.Content, &article.Redirect)

	if err == nil {
		return &article, nil
	}

	// Try case-insensitive match
	caser := cases.Title(language.AmericanEnglish)
	titleCase := caser.String(strings.ToLower(title))
	err = w.db.QueryRow(`
		SELECT id, title, namespace, content, redirect
		FROM articles
		WHERE LOWER(title) = LOWER(?)
		LIMIT 1
	`, titleCase).Scan(&article.ID, &article.Title, &article.Namespace, &article.Content, &article.Redirect)

	if err != nil {
		return nil, fmt.Errorf("article not found: %s", title)
	}

	return &article, nil
}

// SearchTitles searches for article titles using FTS or LIKE queries
func (w *Wiki) SearchTitles(query string, limit int) ([]string, error) {
	if err := w.Open(); err != nil {
		return nil, err
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if limit <= 0 {
		limit = 20
	}

	var rows *sql.Rows
	var err error

	// Use FTS if available, otherwise fall back to LIKE
	if w.ftsVersion == "fts5" || w.ftsVersion == "fts4" {
		// Build FTS query - escape special characters and add prefix matching
		// FTS special characters: ", ', \
		escapedQuery := strings.ReplaceAll(query, `"`, `""`)
		escapedQuery = strings.ReplaceAll(escapedQuery, `'`, `''`)
		// For prefix matching, add * to the end of the query
		ftsQuery := escapedQuery + "*"

		rows, err = w.db.Query(`
			SELECT DISTINCT title
			FROM articles_fts
			WHERE articles_fts MATCH ?
			ORDER BY rank
			LIMIT ?
		`, ftsQuery, limit)

		if err != nil {
			// FTS query failed, fall back to LIKE
			log.Printf("FTS query failed, falling back to LIKE: %v", err)
			w.ftsVersion = "none"
		}
	}

	// If FTS is not available or failed, use LIKE query
	if w.ftsVersion == "none" || rows == nil {
		rows, err = w.db.Query(`
			SELECT DISTINCT title
			FROM articles
			WHERE title LIKE ?
			ORDER BY title
			LIMIT ?
		`, "%"+query+"%", limit)
		if err != nil {
			return nil, fmt.Errorf("search failed: %w", err)
		}
	}
	defer rows.Close()

	var titles []string
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			continue
		}
		titles = append(titles, title)
	}

	return titles, nil
}

// GetArticleByID retrieves an article by ID
func (w *Wiki) GetArticleByID(id int64) (*Article, error) {
	if err := w.Open(); err != nil {
		return nil, err
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	var article Article
	err := w.db.QueryRow(`
		SELECT id, title, namespace, content, redirect
		FROM articles
		WHERE id = ?
	`, id).Scan(&article.ID, &article.Title, &article.Namespace, &article.Content, &article.Redirect)

	if err != nil {
		return nil, fmt.Errorf("article not found: %d", id)
	}

	return &article, nil
}

// Page represents a Wikipedia page in XML format
type Page struct {
	XMLName    xml.Name   `xml:"page"`
	Title      string     `xml:"title"`
	NS         int        `xml:"ns"`
	ID         int        `xml:"id"`
	Redirect   []Redirect `xml:"redirect"`
	RevisionID string     `xml:"revision>id"`
	Timestamp  string     `xml:"revision>timestamp"`
	Username   string     `xml:"revision>contributor>username"`
	UserID     string     `xml:"revision>contributor>id"`
	Model      string     `xml:"revision>model"`
	Format     string     `xml:"revision>format"`
	Text       string     `xml:"revision>text"`
}

type Redirect struct {
	Title string `xml:"title,attr"`
}
