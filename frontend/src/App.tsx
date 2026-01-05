import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { SearchBox } from './components/SearchBox';
import { SearchResults } from './components/SearchResults';
import { ArticleView } from './components/ArticleView';
import { searchArticles, getArticle } from './utils/api';
import type { SearchResponse, Article } from './types';
import './App.css';

function App() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<SearchResponse | null>(null);
  const [article, setArticle] = useState<Article | null>(null);
  const [isSearching, setIsSearching] = useState(false);
  const [isLoadingArticle, setIsLoadingArticle] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [view, setView] = useState<'search' | 'article'>('search');

  // Initialize from URL parameters
  useEffect(() => {
    const articleParam = searchParams.get('article');
    const queryParam = searchParams.get('q');

    if (articleParam) {
      loadArticle(articleParam, false);
    } else if (queryParam) {
      setSearchQuery(queryParam);
      performSearch(queryParam, false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const performSearch = async (query: string, updateUrl: boolean = true) => {
    setIsSearching(true);
    setError(null);
    setArticle(null);
    setView('search');
    setSearchQuery(query);

    if (updateUrl) {
      navigate(`?q=${encodeURIComponent(query)}`, { replace: false });
    }

    try {
      const results = await searchArticles(query, 20);
      setSearchResults(results);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Search failed';
      setError(message);
      setSearchResults(null);
    } finally {
      setIsSearching(false);
    }
  };

  const loadArticle = async (title: string, updateUrl: boolean = true) => {
    setIsLoadingArticle(true);
    setError(null);
    setSearchResults(null);
    setView('article');

    if (updateUrl) {
      navigate(`?article=${encodeURIComponent(title)}`, { replace: false });
    }

    try {
      const articleData = await getArticle(title);
      setArticle(articleData);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load article';
      setError(message);
      setArticle(null);
    } finally {
      setIsLoadingArticle(false);
    }
  };

  const handleBack = () => {
    if (searchQuery) {
      navigate(`?q=${encodeURIComponent(searchQuery)}`, { replace: false });
      setView('search');
      setArticle(null);
    } else {
      navigate('/', { replace: false });
      setView('search');
      setArticle(null);
      setSearchResults(null);
    }
  };

  return (
    <>
      <div className="header">
        <h1>📚 Wikipedia SQLite</h1>
        <p>Search and browse Wikipedia articles</p>
      </div>

      <SearchBox
        onSearch={performSearch}
        initialQuery={searchQuery}
        isLoading={isSearching}
      />

      {error && view === 'search' && <div className="error">{error}</div>}

      {view === 'search' && searchResults && (
        <SearchResults
          data={searchResults}
          onArticleClick={loadArticle}
        />
      )}

      {view === 'article' && (
        <ArticleView
          article={article}
          isLoading={isLoadingArticle}
          error={error}
          onBack={handleBack}
        />
      )}
    </>
  );
}

export default App;

