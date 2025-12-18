import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { assetService } from '../services/assetService';
import { RoleBadge } from '../components/RoleBadge';
import type { Realm, CreateRealmData } from '../types';
import { FolderOpen, Plus, AlertCircle, X, Filter } from 'lucide-react';

export function RealmsListPage() {
  const [realms, setRealms] = useState<Realm[]>([]);
  const [filteredRealms, setFilteredRealms] = useState<Realm[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [roleFilter, setRoleFilter] = useState<'all' | 'admin' | 'member'>('all');
  const [formData, setFormData] = useState<CreateRealmData>({ name: '', description: '' });
  const [formError, setFormError] = useState('');

  useEffect(() => {
    loadRealms();
  }, []);

  useEffect(() => {
    if (roleFilter === 'all') {
      setFilteredRealms(realms);
    } else {
      setFilteredRealms(realms.filter((r) => r.role === roleFilter));
    }
  }, [realms, roleFilter]);

  const loadRealms = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await assetService.getRealms();
      setRealms(data);
      setFilteredRealms(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load realms');
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError('');

    if (!formData.name.trim()) {
      setFormError('Name is required');
      return;
    }

    try {
      await assetService.createRealm(formData);
      setShowCreateModal(false);
      setFormData({ name: '', description: '' });
      await loadRealms();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create realm');
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
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">My Bars</h1>
            <p className="text-gray-600 mt-2">Manage your bars and collaborate with bartenders</p>
          </div>
          <button
            onClick={() => setShowCreateModal(true)}
            className="inline-flex items-center gap-2 px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors shadow-lg"
          >
            <Plus className="h-4 w-4" />
            Create Bar
          </button>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg flex items-center gap-2 text-red-800">
            <AlertCircle className="h-5 w-5" />
            <span>{error}</span>
          </div>
        )}

        <div className="mb-6 flex items-center gap-4">
          <div className="flex items-center gap-2">
            <Filter className="h-5 w-5 text-gray-600" />
            <span className="text-sm font-medium text-gray-700">Filter by role:</span>
          </div>
          <div className="flex gap-2">
            {(['all', 'admin', 'member'] as const).map((filter) => {
              const displayName = filter === 'admin' ? 'Manager' : filter === 'member' ? 'Bartender' : 'All';
              return (
                <button
                  key={filter}
                  onClick={() => setRoleFilter(filter)}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                    roleFilter === filter
                      ? 'bg-orange-600 text-white'
                      : 'bg-white text-gray-700 hover:bg-gray-50'
                  }`}
                >
                  {displayName}
                </button>
              );
            })}
          </div>
        </div>

        {filteredRealms.length === 0 ? (
          <div className="bg-white rounded-lg shadow p-12 text-center">
            <FolderOpen className="h-16 w-16 text-gray-400 mx-auto mb-4" />
            <p className="text-gray-600 mb-4">
              {roleFilter === 'all' ? 'No bars yet' : `No ${roleFilter} bars`}
            </p>
            {roleFilter === 'all' && (
              <button
                onClick={() => setShowCreateModal(true)}
                className="inline-flex items-center gap-2 px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors"
              >
                <Plus className="h-4 w-4" />
                Create Your First Bar
              </button>
            )}
          </div>
        ) : (
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredRealms.map((realm) => (
              <Link
                key={realm.id}
                to={`/realms/${realm.id}`}
                className="bg-white rounded-lg shadow hover:shadow-lg transition-all p-6 border border-gray-200 hover:border-orange-300"
              >
                <div className="flex items-start justify-between mb-4">
                  <div className="p-2 bg-orange-100 rounded-lg">
                    <FolderOpen className="h-6 w-6 text-orange-600" />
                  </div>
                  <RoleBadge role={realm.role!} />
                </div>
                <h3 className="text-lg font-semibold text-gray-900 mb-2">{realm.name}</h3>
                <p className="text-sm text-gray-600 mb-4 line-clamp-2">
                  {realm.description || 'No description'}
                </p>
                <p className="text-xs text-gray-500">
                  Created {new Date(realm.created_at).toLocaleDateString()}
                </p>
              </Link>
            ))}
          </div>
        )}

        {showCreateModal && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg shadow-xl max-w-md w-full p-6">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-2xl font-bold text-gray-900">Create Bar</h2>
                <button
                  onClick={() => {
                    setShowCreateModal(false);
                    setFormData({ name: '', description: '' });
                    setFormError('');
                  }}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <X className="h-6 w-6" />
                </button>
              </div>

              {formError && (
                <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-center gap-2 text-red-800 text-sm">
                  <AlertCircle className="h-4 w-4" />
                  <span>{formError}</span>
                </div>
              )}

              <form onSubmit={handleCreate} className="space-y-4">
                <div>
                  <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-1">
                    Bar Name *
                  </label>
                  <input
                    id="name"
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    placeholder="Downtown Bar"
                    required
                  />
                </div>

                <div>
                  <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-1">
                    Description
                  </label>
                  <textarea
                    id="description"
                    value={formData.description}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    rows={3}
                    placeholder="Description of the bar..."
                  />
                </div>

                <div className="flex gap-3">
                  <button
                    type="submit"
                    className="flex-1 py-2 bg-orange-600 text-white rounded-lg font-semibold hover:bg-orange-700 transition-colors"
                  >
                    Create
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setShowCreateModal(false);
                      setFormData({ name: '', description: '' });
                      setFormError('');
                    }}
                    className="flex-1 py-2 bg-gray-200 text-gray-700 rounded-lg font-semibold hover:bg-gray-300 transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}