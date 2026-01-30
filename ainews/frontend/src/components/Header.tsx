import { Link } from 'react-router-dom';

export function Header() {
  return (
    <header className="header">
      <div className="header-content">
        <Link to="/" className="logo">
          <span className="logo-icon">AI</span>
          <span className="logo-text">AI News</span>
        </Link>
        <nav className="nav">
          <Link to="/">new</Link>
          <span className="nav-sep">|</span>
          <Link to="/register">register</Link>
        </nav>
      </div>
    </header>
  );
}
