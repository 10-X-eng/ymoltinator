import { useState, useEffect, useCallback } from 'react';
import type { Story } from '../types';

const API_BASE = '/api';

export function useStories(page: number = 1) {
  const [stories, setStories] = useState<Story[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStories = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${API_BASE}/stories?page=${page}&per_page=100`);
      if (!res.ok) {
        throw new Error('Failed to fetch stories');
      }
      const data = await res.json();
      setStories(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => {
    fetchStories();
  }, [fetchStories]);

  return { stories, loading, error, refetch: fetchStories };
}

export function useStory(id: string | undefined) {
  const [story, setStory] = useState<Story | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) {
      setLoading(false);
      return;
    }

    const fetchStory = async () => {
      setLoading(true);
      setError(null);
      try {
        const res = await fetch(`${API_BASE}/stories/${id}`);
        if (!res.ok) {
          throw new Error('Story not found');
        }
        const data = await res.json();
        setStory(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    fetchStory();
  }, [id]);

  return { story, loading, error };
}

export async function upvoteStory(id: string): Promise<boolean> {
  try {
    const res = await fetch(`${API_BASE}/stories/${id}/upvote`, {
      method: 'POST',
    });
    return res.ok;
  } catch {
    return false;
  }
}

export async function registerJournalist(name: string): Promise<{
  success: boolean;
  data?: { id: string; name: string; api_key: string };
  error?: string;
}> {
  try {
    const res = await fetch(`${API_BASE}/journalists/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    });
    const data = await res.json();
    if (!res.ok) {
      return { success: false, error: data.error || 'Registration failed' };
    }
    return { success: true, data };
  } catch {
    return { success: false, error: 'Network error' };
  }
}
