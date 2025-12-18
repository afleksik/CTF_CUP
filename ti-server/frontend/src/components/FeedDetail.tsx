import { useEffect, useState } from 'react';
import { ArrowLeft, Shield, AlertTriangle, Plus, Key } from 'lucide-react';
import { tiService } from '../services/tiService';
import type { Feed, Indicator } from '../types';
import { IOCList } from './IOCList';
import { AddIOCForm } from './AddIOCForm';

interface FeedDetailProps {
  feed: Feed;
  onBack: () => void;
}

export function FeedDetail({ feed, onBack }: FeedDetailProps) {
  const [iocs, setIOCs] = useState<Indicator[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showAddForm, setShowAddForm] = useState(false);
  const [apiKey, setApiKey] = useState('');

  useEffect(() => {
    loadIOCs();
  }, [feed.id]);

  const loadIOCs = async () => {
    try {
      setLoading(true);
      setError(null);
      const iocs = await tiService.getFeedIndicators(feed.id);
      setIOCs(iocs);
    } catch (err) {
      setError('Failed to load IOCs');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleAddIOC = async (iocData: { type: string; value: string; severity: 'low' | 'medium' | 'high' | 'critical'; description: string }) => {
    try {
      if (!feed.is_public && !apiKey) {
        alert('API key is required to add IOCs to private feeds');
        return;
      }
      await tiService.createIndicator({
        feed_id: feed.id,
        ...iocData
      }, apiKey || '');
      await loadIOCs();
      setShowAddForm(false);
    } catch (err) {
      console.error('Failed to add IOC:', err);
      alert('Failed to add IOC. ' + (feed.is_public ? '' : 'Make sure you have a valid API key.'));
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <button
          onClick={onBack}
          className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
        >
          <ArrowLeft className="h-6 w-6 text-gray-600" />
        </button>
        <div className="flex-1">
          <h2 className="text-3xl font-bold text-gray-900">{feed.name}</h2>
          <p className="text-gray-600 mt-1">{feed.description}</p>
        </div>
        <Shield className="h-10 w-10 text-blue-600" />
      </div>

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-gray-500">Status:</span>
            <span className={`ml-2 font-medium ${feed.is_public ? 'text-green-600' : 'text-gray-600'}`}>
              {feed.is_public ? 'Public' : 'Private'}
            </span>
          </div>
          <div>
            <span className="text-gray-500">Created:</span>
            <span className="ml-2 font-medium text-gray-900">
              {new Date(feed.created_at).toLocaleDateString()}
            </span>
          </div>
        </div>
      </div>

      {!feed.is_public && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
          <div className="flex items-start gap-3">
            <Key className="h-5 w-5 text-yellow-600 mt-0.5 flex-shrink-0" />
            <div className="flex-1">
              <h4 className="font-medium text-yellow-900 mb-2">API Key Required</h4>
              <input
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder="Enter your API key to access private feed data"
                className="w-full px-3 py-2 border border-yellow-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-yellow-500"
              />
            </div>
          </div>
        </div>
      )}

      <div className="flex items-center justify-between">
        <h3 className="text-xl font-bold text-gray-900 flex items-center gap-2">
          <AlertTriangle className="h-6 w-6 text-orange-500" />
          Indicators of Compromise
        </h3>
        <button
          onClick={() => setShowAddForm(!showAddForm)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          <Plus className="h-5 w-5" />
          Add IOC
        </button>
      </div>

      {showAddForm && (
        <AddIOCForm
          onSubmit={handleAddIOC}
          onCancel={() => setShowAddForm(false)}
        />
      )}

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      ) : error ? (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <p className="text-red-800">{error}</p>
        </div>
      ) : (
        <IOCList iocs={iocs} />
      )}
    </div>
  );
}
