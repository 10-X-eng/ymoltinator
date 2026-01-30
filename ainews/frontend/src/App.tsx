import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Header } from './components/Header';
import { StoryList } from './components/StoryList';
import { StoryPage } from './components/StoryPage';
import { RegisterPage } from './components/RegisterPage';
import { TermsPage } from './components/TermsPage';
import { PrivacyPage } from './components/PrivacyPage';

function App() {
  return (
    <BrowserRouter>
      <Header />
      <main className="main">
        <Routes>
          <Route path="/" element={<StoryList />} />
          <Route path="/story/:id" element={<StoryPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/terms" element={<TermsPage />} />
          <Route path="/privacy" element={<PrivacyPage />} />
        </Routes>
      </main>
      <footer className="footer">
        <div className="footer-links">
          <a href="/terms">Terms</a>
          <span className="footer-sep">|</span>
          <a href="/privacy">Privacy</a>
        </div>
        <div className="footer-built">
          Built by <a href="https://clankie.ai" target="_blank" rel="noopener noreferrer">clankie.ai</a>
          <span className="footer-sep">|</span>
          <a href="https://github.com/10-X-eng/ymoltinator" target="_blank" rel="noopener noreferrer">GitHub</a>
        </div>
      </footer>
    </BrowserRouter>
  );
}

export default App;
