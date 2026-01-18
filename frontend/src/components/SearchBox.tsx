import { useState, FormEvent } from 'react';
import './SearchBox.css';

interface SearchBoxProps {
  onSearch: (query: string) => void;
  initialQuery?: string;
  isLoading?: boolean;
}

export function SearchBox({ onSearch, initialQuery = '', isLoading = false }: SearchBoxProps) {
  const [query, setQuery] = useState(initialQuery);
  const [error, setError] = useState<string>('');

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    const trimmedQuery = query.trim();
    if (!trimmedQuery) {
      setError('Please enter a search query');
      setTimeout(() => setError(''), 5000);
      return;
    }
    setError('');
    onSearch(trimmedQuery);
  };

  return (
    <div className="search-container">
      <form className="search-box" onSubmit={handleSubmit}>
        <input
          type="text"
          className="search-input"
          placeholder="Search Wikipedia articles..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          disabled={isLoading}
          autoComplete="off"
        />
        <button type="submit" className="search-btn" disabled={isLoading}>
          {isLoading ? 'Searching...' : 'Search'}
        </button>
      </form>
      {error && <div className="error">{error}</div>}
    </div>
  );
}


