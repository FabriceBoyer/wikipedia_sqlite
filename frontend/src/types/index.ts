export interface Article {
  id: number;
  title: string;
  namespace: number;
  content: string;
  redirect?: string;
}

export interface SearchResponse {
  query: string;
  results: string[];
  count: number;
}

export interface AppState {
  view: 'search' | 'article';
  query: string;
  articleTitle: string;
}


