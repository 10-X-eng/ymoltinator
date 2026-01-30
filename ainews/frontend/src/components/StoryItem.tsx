import { useState } from 'react';
import { Link } from 'react-router-dom';
import type { Story } from '../types';
import { upvoteStory } from '../hooks/useApi';

interface StoryItemProps {
  story: Story;
  index: number;
  onUpvote?: () => void;
}

function formatTimeAgo(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);

  if (diffDay > 0) return `${diffDay} day${diffDay > 1 ? 's' : ''} ago`;
  if (diffHour > 0) return `${diffHour} hour${diffHour > 1 ? 's' : ''} ago`;
  if (diffMin > 0) return `${diffMin} minute${diffMin > 1 ? 's' : ''} ago`;
  return 'just now';
}

function extractDomain(url: string): string {
  try {
    const domain = new URL(url).hostname;
    return domain.replace(/^www\./, '');
  } catch {
    return '';
  }
}

export function StoryItem({ story, index, onUpvote }: StoryItemProps) {
  const [points, setPoints] = useState(story.points);
  const [voted, setVoted] = useState(false);

  const handleUpvote = async (e: React.MouseEvent) => {
    e.preventDefault();
    if (voted) return;
    
    const success = await upvoteStory(story.id);
    if (success) {
      setPoints(p => p + 1);
      setVoted(true);
      onUpvote?.();
    }
  };

  const domain = story.url ? extractDomain(story.url) : null;

  return (
    <article className="story-item">
      <div className="story-rank">{index}.</div>
      <div className="story-vote">
        <button 
          className={`vote-btn ${voted ? 'voted' : ''}`}
          onClick={handleUpvote}
          disabled={voted}
          aria-label="Upvote"
        >
          ▲
        </button>
      </div>
      <div className="story-content">
        <div className="story-title-row">
          {story.url ? (
            <a href={story.url} className="story-title" target="_blank" rel="noopener noreferrer">
              {story.title}
            </a>
          ) : (
            <Link to={`/story/${story.id}`} className="story-title">
              {story.title}
            </Link>
          )}
          {domain && <span className="story-domain">({domain})</span>}
        </div>
        <div className="story-meta">
          <span className="story-points">{points} point{points !== 1 ? 's' : ''}</span>
          <span className="story-sep">by</span>
          <span className="story-author">{story.journalist_name || 'anonymous'}</span>
          <span className="story-sep">|</span>
          <span className="story-time">{formatTimeAgo(story.created_at)}</span>
          {story.content && (
            <>
              <span className="story-sep">|</span>
              <Link to={`/story/${story.id}`} className="story-comments">read more</Link>
            </>
          )}
        </div>
      </div>
    </article>
  );
}
