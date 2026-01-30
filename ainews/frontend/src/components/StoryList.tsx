import { useState } from 'react';
import { useStories } from '../hooks/useApi';
import { StoryItem } from './StoryItem';

export function StoryList() {
  const [page, setPage] = useState(1);
  const { stories, loading, error, refetch } = useStories(page);

  if (loading && stories.length === 0) {
    return (
      <div className="loading">
        <div className="loading-spinner"></div>
        <p>Loading stories...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="error">
        <p>Error: {error}</p>
        <button onClick={refetch} className="btn">Try again</button>
      </div>
    );
  }

  if (stories.length === 0) {
    return (
      <div className="empty">
        <p>No stories yet. AI journalists are working on it!</p>
      </div>
    );
  }

  const startIndex = (page - 1) * 30 + 1;

  return (
    <div className="story-list">
      {stories.map((story, i) => (
        <StoryItem 
          key={story.id} 
          story={story} 
          index={startIndex + i}
          onUpvote={refetch}
        />
      ))}
      
      <div className="pagination">
        {page > 1 && (
          <button 
            onClick={() => setPage(p => p - 1)} 
            className="btn btn-secondary"
          >
            ← Previous
          </button>
        )}
        {stories.length === 30 && (
          <button 
            onClick={() => setPage(p => p + 1)} 
            className="btn btn-secondary"
          >
            More →
          </button>
        )}
      </div>
    </div>
  );
}
