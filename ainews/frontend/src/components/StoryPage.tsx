import { useParams, Link } from 'react-router-dom';
import { useStory } from '../hooks/useApi';
import { renderTextWithLinks, normalizeUrl } from '../utils/linkify';

export function StoryPage() {
  const { id } = useParams<{ id: string }>();
  const { story, loading, error } = useStory(id);

  if (loading) {
    return (
      <div className="loading">
        <div className="loading-spinner"></div>
        <p>Loading story...</p>
      </div>
    );
  }

  if (error || !story) {
    return (
      <div className="error">
        <p>Story not found</p>
        <Link to="/" className="btn">Back to home</Link>
      </div>
    );
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  return (
    <div className="story-page">
      <article className="story-full">
        <h1 className="story-full-title">
          {story.url ? (
            <a href={normalizeUrl(story.url)} target="_blank" rel="noopener noreferrer">
              {story.title}
            </a>
          ) : (
            renderTextWithLinks(story.title)
          )}
        </h1>
        
        <div className="story-full-meta">
          <span>{story.points} point{story.points !== 1 ? 's' : ''}</span>
          <span>by {story.journalist_name || 'anonymous'}</span>
          <span>on {formatDate(story.created_at)}</span>
        </div>

        {story.url && (
          <p className="story-full-link">
            <a href={normalizeUrl(story.url)} target="_blank" rel="noopener noreferrer">
              {story.url}
            </a>
          </p>
        )}

        {story.content && (
          <div className="story-full-content">
            {story.content.split('\n').map((paragraph, i) => (
              <p key={i}>{renderTextWithLinks(paragraph)}</p>
            ))}
          </div>
        )}
      </article>

      <Link to="/" className="back-link">← Back to stories</Link>
    </div>
  );
}
