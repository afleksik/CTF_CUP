import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { assetService } from '../services/assetService';
import { RoleBadge } from '../components/RoleBadge';
import type { Realm } from '../types';
import { FolderOpen, Package, Plus, AlertCircle } from 'lucide-react';

export function DashboardPage() {
  const { user } = useAuth();
  const [realms, setRealms] = useState<Realm[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadRealms();
  }, []);

  const loadRealms = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await assetService.getRealms();
      setRealms(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load realms');
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-orange-600"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-orange-50 to-amber-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900">
            Welcome back, {user?.username}!
          </h1>
          <p className="text-gray-600 mt-2">
            Manage your bars and inventory from this dashboard
          </p>
        </div>

        <div className="grid md:grid-cols-3 gap-6 mb-8">
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">Total Bars</p>
                <p className="text-3xl font-bold text-gray-900">{realms.length}</p>
              </div>
              <FolderOpen className="h-12 w-12 text-orange-600 opacity-20" />
            </div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">Managed Bars</p>
                <p className="text-3xl font-bold text-gray-900">
                  {realms.filter((r) => r.role === 'admin').length}
                </p>
              </div>
              <Package className="h-12 w-12 text-blue-600 opacity-20" />
            </div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">Bartender At</p>
                <p className="text-3xl font-bold text-gray-900">
                  {realms.filter((r) => r.role === 'member').length}
                </p>
              </div>
              <Package className="h-12 w-12 text-green-600 opacity-20" />
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow">
          <div className="p-6 border-b border-gray-200">
            <div className="flex items-center justify-between">
              <h2 className="text-xl font-semibold text-gray-900">Recent Bars</h2>
              <Link
                to="/realms"
                className="inline-flex items-center gap-2 px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors"
              >
                <Plus className="h-4 w-4" />
                View All Bars
              </Link>
            </div>
          </div>

          {error && (
            <div className="p-4 bg-red-50 border-b border-red-200 flex items-center gap-2 text-red-800">
              <AlertCircle className="h-5 w-5" />
              <span>{error}</span>
            </div>
          )}

          <div className="p-6">
            {realms.length === 0 ? (
              <div className="text-center py-12">
                <FolderOpen className="h-16 w-16 text-gray-400 mx-auto mb-4" />
                <p className="text-gray-600 mb-4">No bars yet</p>
                <Link
                  to="/realms"
                  className="inline-flex items-center gap-2 px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors"
                >
                  <Plus className="h-4 w-4" />
                  Create Your First Bar
                </Link>
              </div>
            ) : (
              <div className="grid gap-4">
                {realms.slice(0, 5).map((realm) => (
                  <Link
                    key={realm.id}
                    to={`/realms/${realm.id}`}
                    className="flex items-center justify-between p-4 border border-gray-200 rounded-lg hover:border-orange-300 hover:bg-orange-50 transition-all"
                  >
                    <div className="flex items-center gap-4">
                      <div className="p-2 bg-orange-100 rounded-lg">
                        <FolderOpen className="h-6 w-6 text-orange-600" />
                      </div>
                      <div>
                        <h3 className="font-semibold text-gray-900">{realm.name}</h3>
                        <p className="text-sm text-gray-600">{realm.description || 'No description'}</p>
                      </div>
                    </div>
                    <RoleBadge role={realm.role!} />
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}