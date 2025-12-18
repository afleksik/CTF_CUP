import { useState } from 'react';
import { FeedList } from './components/FeedList';
import { FeedDetail } from './components/FeedDetail';
import type { Feed } from './types';

function App() {
  const [selectedFeed, setSelectedFeed] = useState<Feed | null>(null);

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <header className="mb-8">
          <div className="flex items-center gap-3">
            <div className="p-3 bg-red-600 rounded-lg shadow-lg">
              <svg
                className="h-8 w-8 text-white"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <circle cx="12" cy="12" r="10" strokeWidth={2} />
                <circle cx="12" cy="12" r="6" strokeWidth={2} />
                <circle cx="12" cy="12" r="2" strokeWidth={2} fill="currentColor" />
                <line x1="12" y1="2" x2="12" y2="7" strokeWidth={2} />
                <line x1="12" y1="17" x2="12" y2="22" strokeWidth={2} />
                <line x1="2" y1="12" x2="7" y2="12" strokeWidth={2} />
                <line x1="17" y1="12" x2="22" y2="12" strokeWidth={2} />
              </svg>
            </div>
            <div>
              <h1 className="text-3xl font-bold text-gray-900">
                Hazy Threat Intelligence Server
              </h1>
              <p className="text-gray-600 mt-1">
                Monitor and manage security threat feeds
              </p>
            </div>
          </div>
        </header>

        <main>
          {selectedFeed ? (
            <FeedDetail
              feed={selectedFeed}
              onBack={() => setSelectedFeed(null)}
            />
          ) : (
            <FeedList onSelectFeed={setSelectedFeed} />
          )}
        </main>

        <footer className="mt-16 pt-8 pb-6 border-t border-gray-200">
          <div className="text-center text-gray-600">
            <p className="text-sm">
              © {new Date().getFullYear()} <span className="font-semibold text-gray-900">HazyCorp Team</span>. All rights reserved.
            </p>
          </div>
        </footer>
      </div>
    </div>
  );
}

export default App;
