import { SearchResponse } from '../types';
import './SearchResults.css';

interface SearchResultsProps {
  data: SearchResponse | null;
  onArticleClick: (title: string) => void;
}

export function SearchResults({ data, onArticleClick }: SearchResultsProps) {
  if (!data) {
    return null;
  }

  return (
    <div className="results-container">
      <div className="results-header">
        Found {data.count} result{data.count !== 1 ? 's' : ''} for "{data.query}"
      </div>
      {data.results.length === 0 ? (
        <div className="loading">No results found</div>
      ) : (
        <ul className="results-list">
          {data.results.map((title, index) => (
            <li
              key={`${title}-${index}`}
              className="result-item"
              onClick={() => onArticleClick(title)}
            >
              <div className="result-item-title">{title}</div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

