import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { assetService } from '../services/assetService';
import { AssetTypeBadge } from '../components/AssetTypeBadge';
import type { Asset, Realm, AssetType } from '../types';
import { ArrowLeft, Edit, Trash2, AlertCircle, FolderOpen, X } from 'lucide-react';

export function AssetDetailPage() {
  const { assetId } = useParams<{ assetId: string }>();
  const { user } = useAuth();
  const navigate = useNavigate();

  const [asset, setAsset] = useState<Asset | null>(null);
  const [realm, setRealm] = useState<Realm | null>(null);
  const [ownerUsername, setOwnerUsername] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Edit asset state
  const [showEditModal, setShowEditModal] = useState(false);
  const [editFormData, setEditFormData] = useState({
    name: '',
    asset_type: 'spirits' as AssetType,
    description: '',
    owner_user_id: '',
  });
  const [editError, setEditError] = useState('');
  const [realmUsers, setRealmUsers] = useState<Array<{ user_id: string; username?: string }>>([]);

  useEffect(() => {
    if (assetId) {
      loadAsset();
    }
  }, [assetId]);

  const loadAsset = async () => {
    try {
      setLoading(true);
      setError(null);
      const assetData = await assetService.getAsset(assetId!);
      setAsset(assetData);

      // Load realm to check permissions
      const realmData = await assetService.getRealm(assetData.realm_id);
      setRealm(realmData);

      // Load owner username
      try {
        const ownerInfo = await assetService.getUserInfo(assetData.owner_user_id);
        setOwnerUsername(ownerInfo.username);
      } catch (err) {
        console.error('Failed to load owner info:', err);
        setOwnerUsername(assetData.owner_user_id); // Fallback to ID
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load asset');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this asset? This action cannot be undone.')) {
      return;
    }

    try {
      await assetService.deleteAsset(assetId!);
      navigate(`/realms/${asset!.realm_id}`);
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete asset');
    }
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    setEditError('');

    try {
      const updatedAsset = await assetService.updateAsset(assetId!, {
        ...editFormData,
      });
      setAsset(updatedAsset);

      // Update owner username if owner changed
      if (updatedAsset.owner_user_id !== asset?.owner_user_id) {
        try {
          const ownerInfo = await assetService.getUserInfo(updatedAsset.owner_user_id);
          setOwnerUsername(ownerInfo.username);
        } catch (err) {
          console.error('Failed to load new owner info:', err);
          setOwnerUsername(updatedAsset.owner_user_id);
        }
      }

      setShowEditModal(false);
    } catch (err) {
      setEditError(err instanceof Error ? err.message : 'Failed to update asset');
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-orange-600"></div>
      </div>
    );
  }

  if (error || !asset || !realm) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <AlertCircle className="h-16 w-16 text-red-500 mx-auto mb-4" />
          <p className="text-xl text-gray-900 mb-2">Failed to load asset</p>
          <p className="text-gray-600 mb-4">{error}</p>
          <Link to="/realms" className="text-orange-600 hover:text-orange-700">
            Back to Realms
          </Link>
        </div>
      </div>
    );
  }

  const canEdit = realm.role === 'admin' || asset.owner_user_id === user?.id;

  return (
    <div className="min-h-screen bg-gradient-to-br from-orange-50 to-amber-50">
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-6">
          <Link
            to={`/realms/${asset.realm_id}`}
            className="inline-flex items-center gap-2 text-gray-600 hover:text-gray-900 mb-4"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to Realm
          </Link>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-start justify-between mb-6">
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-2">
                  <h1 className="text-3xl font-bold text-gray-900">{asset.name}</h1>
                  <AssetTypeBadge type={asset.asset_type} />
                </div>
                <p className="text-gray-600">{asset.description || 'No description'}</p>
              </div>
              {canEdit && (
                <div className="flex gap-2">
                  <button
                    onClick={async () => {
                      setEditFormData({
                        name: asset.name,
                        asset_type: asset.asset_type,
                        description: asset.description || '',
                        owner_user_id: asset.owner_user_id,
                      });

                      // Load realm users for owner selection
                      try {
                        const usersData = await assetService.getRealmUsers(asset.realm_id, 1000, 0);
                        setRealmUsers(usersData.data);
                      } catch (err) {
                        console.error('Failed to load realm users:', err);
                      }

                      setShowEditModal(true);
                    }}
                    className="p-2 text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
                  >
                    <Edit className="h-5 w-5" />
                  </button>
                  <button
                    onClick={handleDelete}
                    className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                  >
                    <Trash2 className="h-5 w-5" />
                  </button>
                </div>
              )}
            </div>

            <div className="grid md:grid-cols-2 gap-6 mb-6">
              <div>
                <h3 className="text-sm font-medium text-gray-500 mb-1">Asset Type</h3>
                <p className="text-gray-900 capitalize">{asset.asset_type}</p>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-500 mb-1">Owner</h3>
                <p className="text-gray-900">
                  {ownerUsername ? `${ownerUsername} (${asset.owner_user_id})` : asset.owner_user_id}
                </p>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-500 mb-1">Created At</h3>
                <p className="text-gray-900">
                  {new Date(asset.created_at).toLocaleString()}
                </p>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-500 mb-1">Updated At</h3>
                <p className="text-gray-900">
                  {new Date(asset.updated_at).toLocaleString()}
                </p>
              </div>
            </div>

            <div className="mb-6">
              <h3 className="text-sm font-medium text-gray-500 mb-2">Realm</h3>
              <Link
                to={`/realms/${realm.id}`}
                className="inline-flex items-center gap-2 p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
              >
                <FolderOpen className="h-5 w-5 text-orange-600" />
                <div>
                  <p className="font-medium text-gray-900">{realm.name}</p>
                  <p className="text-sm text-gray-600">{realm.description || 'No description'}</p>
                </div>
              </Link>
            </div>

          </div>
        </div>

        {/* Edit Asset Modal */}
        {showEditModal && canEdit && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg shadow-xl max-w-lg w-full p-6 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-2xl font-bold text-gray-900">Edit Asset</h2>
                <button
                  onClick={() => {
                    setShowEditModal(false);
                    setEditError('');
                  }}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <X className="h-6 w-6" />
                </button>
              </div>

              {editError && (
                <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-center gap-2 text-red-800 text-sm">
                  <AlertCircle className="h-4 w-4" />
                  <span>{editError}</span>
                </div>
              )}

              <form onSubmit={handleEdit} className="space-y-4">
                <div>
                  <label htmlFor="asset-name" className="block text-sm font-medium text-gray-700 mb-1">
                    Asset Name *
                  </label>
                  <input
                    id="asset-name"
                    type="text"
                    value={editFormData.name}
                    onChange={(e) => setEditFormData({ ...editFormData, name: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    placeholder="Web Server 01"
                    required
                  />
                </div>

                <div>
                  <label htmlFor="asset-type" className="block text-sm font-medium text-gray-700 mb-1">
                    Asset Type *
                  </label>
                  <select
                    id="asset-type"
                    value={editFormData.asset_type}
                    onChange={(e) => setEditFormData({ ...editFormData, asset_type: e.target.value as AssetType })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    required
                  >
                    <option value="spirits">Spirits</option>
                    <option value="wine">Wine</option>
                    <option value="beer">Beer</option>
                    <option value="mixers">Mixers</option>
                    <option value="garnishes">Garnishes</option>
                  </select>
                </div>

                <div>
                  <label htmlFor="asset-description" className="block text-sm font-medium text-gray-700 mb-1">
                    Description
                  </label>
                  <textarea
                    id="asset-description"
                    value={editFormData.description}
                    onChange={(e) => setEditFormData({ ...editFormData, description: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    rows={3}
                    placeholder="Description of the asset..."
                  />
                </div>

                <div>
                  <label htmlFor="asset-owner" className="block text-sm font-medium text-gray-700 mb-1">
                    Owner *
                  </label>
                  <select
                    id="asset-owner"
                    value={editFormData.owner_user_id}
                    onChange={(e) => setEditFormData({ ...editFormData, owner_user_id: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    required
                  >
                    {/* Show current owner even if not in realm (for migration purposes) */}
                    {!realmUsers.find(u => u.user_id === editFormData.owner_user_id) && (
                      <option value={editFormData.owner_user_id}>
                        {ownerUsername || editFormData.owner_user_id} (not in realm - must be changed)
                      </option>
                    )}
                    {/* Show only active realm members */}
                    {realmUsers.map((realmUser) => (
                      <option key={realmUser.user_id} value={realmUser.user_id}>
                        {realmUser.username ? `${realmUser.username} (${realmUser.user_id})` : realmUser.user_id}
                      </option>
                    ))}
                  </select>
                  {!realmUsers.find(u => u.user_id === editFormData.owner_user_id) && (
                    <p className="mt-1 text-xs text-red-600">
                      Current owner is not in the realm. Please select a new owner.
                    </p>
                  )}
                </div>

                <div className="flex gap-3">
                  <button
                    type="submit"
                    className="flex-1 py-2 bg-orange-600 text-white rounded-lg font-semibold hover:bg-orange-700 transition-colors"
                  >
                    Save Changes
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setShowEditModal(false);
                      setEditError('');
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