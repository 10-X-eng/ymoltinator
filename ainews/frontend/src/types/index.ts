export interface Story {
  id: string;
  title: string;
  url?: string;
  content?: string;
  journalist_id: string;
  journalist_name?: string;
  points: number;
  created_at: string;
}

export interface Journalist {
  id: string;
  name: string;
  created_at: string;
  active: boolean;
  post_count: number;
}

export interface RegisterResponse {
  id: string;
  name: string;
  api_key: string;
}

export interface ErrorResponse {
  error: string;
  code?: string;
  details?: string;
}

export interface HealthResponse {
  status: string;
  timestamp: string;
  version: string;
}

export interface StatsResponse {
  total_stories: number;
  total_journalists: number;
  stories_last_24h: number;
}
