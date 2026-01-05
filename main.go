package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/fabriceboyer/common_go_utils/utils"
	"github.com/fabriceboyer/wikipedia_sqlite/wikipedia"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

var wiki *wikipedia.Wiki

func main() {
	// Command line flags
	loadIndex := flag.Bool("load-index", false, "Load the index file into the database")
	processArticles := flag.Bool("process-articles", false, "Process articles from the dump file")
	limit := flag.Int("limit", -1, "Limit the number of entries to process (for testing)")
	flag.Parse()

	err := utils.SetupConfigPath(".")
	if err != nil {
		log.Fatalf("Failed to setup config: %v", err)
	}

	dumpPath := viper.GetString("DUMP_PATH")
	if dumpPath == "" {
		log.Fatal("DUMP_PATH environment variable is not set")
	}

	// Default filenames - can be overridden via env vars
	indexFile := viper.GetString("INDEX_FILE")
	if indexFile == "" {
		indexFile = "articles-multistream-index.txt.bz2"
	}

	articlesFile := viper.GetString("ARTICLES_FILE")
	if articlesFile == "" {
		articlesFile = "articles-multistream.xml.bz2"
	}

	wiki = wikipedia.NewWiki(dumpPath, indexFile, articlesFile)

	// Preprocessing phase
	if *loadIndex {
		log.Println("Loading index...")
		if err := wiki.LoadIndex(*limit); err != nil {
			log.Fatalf("Failed to load index: %v", err)
		}
		log.Println("Index loaded successfully")
	}

	if *processArticles {
		log.Println("Processing articles...")
		if err := wiki.ProcessArticles(*limit); err != nil {
			log.Fatalf("Failed to process articles: %v", err)
		}
		log.Println("Articles processed successfully")
	}

	// If only preprocessing, exit
	if *loadIndex || *processArticles {
		if err := wiki.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
		os.Exit(0)
	}

	// Open database for serving
	if err := wiki.Open(); err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		if err := wiki.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	log.Println("Starting Wikipedia SQLite server...")
	handleRequests()
}

func handleRequests() {
	router := mux.NewRouter().StrictSlash(true)

	// Serve static files
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	router.Handle("/", http.FileServer(http.Dir("./static")))

	// API endpoints
	router.HandleFunc("/api/search", utils.ErrorHandler(handleSearch))
	router.HandleFunc("/api/article", utils.ErrorHandler(handleGetArticle))
	router.HandleFunc("/api/article/{id:[0-9]+}", utils.ErrorHandler(handleGetArticleByID))

	log.Fatal(http.ListenAndServe(":9096", router))
}

func handleSearch(w http.ResponseWriter, r *http.Request) error {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return nil
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}

	titles, err := wiki.SearchTitles(query, limit)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]interface{}{
		"query":  query,
		"results": titles,
		"count":   len(titles),
	})
}

func handleGetArticle(w http.ResponseWriter, r *http.Request) error {
	title := r.URL.Query().Get("title")
	if title == "" {
		http.Error(w, "Missing title parameter", http.StatusBadRequest)
		return nil
	}

	article, err := wiki.GetArticle(title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return nil
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(article)
}

func handleGetArticleByID(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return nil
	}

	article, err := wiki.GetArticleByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return nil
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(article)
}

