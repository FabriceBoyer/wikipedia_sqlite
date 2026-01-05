import type { Article, SearchResponse } from '../types';

const API_BASE = '/api';

export async function searchArticles(query: string, limit: number = 20): Promise<SearchResponse> {
  const response = await fetch(`${API_BASE}/search?q=${encodeURIComponent(query)}&limit=${limit}`);
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  return response.json();
}

export async function getArticle(title: string): Promise<Article> {
  const response = await fetch(`${API_BASE}/article?title=${encodeURIComponent(title)}`);
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  return response.json();
}

export async function getArticleById(id: number): Promise<Article> {
  const response = await fetch(`${API_BASE}/article/${id}`);
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  return response.json();
}

