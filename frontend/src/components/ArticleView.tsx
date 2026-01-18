import { Article } from '../types';
import './ArticleView.css';

interface ArticleViewProps {
  article: Article | null;
  isLoading: boolean;
  error: string | null;
  onBack: () => void;
}

export function ArticleView({ article, isLoading, error, onBack }: ArticleViewProps) {
  if (isLoading) {
    return (
      <div className="article-container">
        <button className="back-btn" onClick={onBack}>
          ← Back to Search
        </button>
        <div className="loading">Loading article...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="article-container">
        <button className="back-btn" onClick={onBack}>
          ← Back to Search
        </button>
        <div className="error">{error}</div>
        <div className="article-header">
          <h2 className="article-title">Error</h2>
        </div>
        <div className="article-content">Failed to load article content</div>
      </div>
    );
  }

  if (!article) {
    return null;
  }

  const content = article.redirect
    ? `This article redirects to: ${article.redirect}`
    : article.content
    ? article.content.length > 5000
      ? article.content.substring(0, 5000) + '\n\n[... content truncated ...]'
      : article.content
    : 'No content available';

  return (
    <div className="article-container">
      <button className="back-btn" onClick={onBack}>
        ← Back to Search
      </button>
      <div className="article-header">
        <h2 className="article-title">{article.title}</h2>
        <div className="article-meta">
          Article ID: {article.id} | Namespace: {article.namespace}
        </div>
      </div>
      <div className="article-content">{content}</div>
    </div>
  );
}


