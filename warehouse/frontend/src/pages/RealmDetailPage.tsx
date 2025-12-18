import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { assetService } from '../services/assetService';
import { RoleBadge } from '../components/RoleBadge';
import { AssetTypeBadge } from '../components/AssetTypeBadge';
import type { Realm, Asset, RealmUser, CreateAssetData, AddUserToRealmData, AssetType, UserSuggestion } from '../types';
import {
  ArrowLeft,
  Edit,
  Trash2,
  Plus,
  Package,
  Users,
  AlertCircle,
  X,
  Search,
  Filter,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';

type TabType = 'assets' | 'users';

export function RealmDetailPage() {
  const { realmId } = useParams<{ realmId: string }>();
  const { user } = useAuth();
  const navigate = useNavigate();

  const [realm, setRealm] = useState<Realm | null>(null);
  const [activeTab, setActiveTab] = useState<TabType>('assets');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Assets state
  const [assets, setAssets] = useState<Asset[]>([]);
  const [assetsTotal, setAssetsTotal] = useState(0);
  const [assetsPage, setAssetsPage] = useState(0);
  const [assetsLimit] = useState(20);
  const [assetTypeFilter, setAssetTypeFilter] = useState<AssetType | ''>('');
  const [assetSearch, setAssetSearch] = useState('');
  const [showCreateAssetModal, setShowCreateAssetModal] = useState(false);
  const [assetFormData, setAssetFormData] = useState<CreateAssetData>({
    name: '',
    asset_type: 'spirits',
    description: '',
    owner_user_id: '',
  });
  const [assetFormError, setAssetFormError] = useState('');
  const [allRealmUsers, setAllRealmUsers] = useState<RealmUser[]>([]);

  // Users state
  const [users, setUsers] = useState<RealmUser[]>([]);
  const [usersTotal, setUsersTotal] = useState(0);
  const [usersPage, setUsersPage] = useState(0);
  const [usersLimit] = useState(20);
  const [showAddUserModal, setShowAddUserModal] = useState(false);
  const [userFormData, setUserFormData] = useState<AddUserToRealmData>({
    user_id: '',
    role: 'member',
  });
  const [userFormError, setUserFormError] = useState('');
  const [userSuggestions, setUserSuggestions] = useState<UserSuggestion[]>([]);
  const [isUserSearchLoading, setIsUserSearchLoading] = useState(false);

  // Edit realm state
  const [showEditRealmModal, setShowEditRealmModal] = useState(false);
  const [realmEditData, setRealmEditData] = useState({ name: '', description: '' });
  const [realmEditError, setRealmEditError] = useState('');

  const isAdmin = realm?.role === 'admin';

  useEffect(() => {
    if (realmId) {
      loadRealm();
    }
  }, [realmId]);

  useEffect(() => {
    if (realm) {
      // Load initial counts for both tabs
      loadAssets();
      loadUsersCount();
    }
  }, [realm]);

  useEffect(() => {
    if (realm && activeTab === 'assets') {
      loadAssets();
    } else if (realm && activeTab === 'users') {
      loadUsers();
    }
  }, [activeTab, assetsPage, assetTypeFilter, assetSearch, usersPage]);

  useEffect(() => {
    if (showCreateAssetModal) {
      loadAllRealmUsers();
    }
  }, [showCreateAssetModal]);

  useEffect(() => {
    if (!showAddUserModal) {
      setUserSuggestions([]);
      setIsUserSearchLoading(false);
      return;
    }

    const query = userFormData.user_id.trim();
    if (query.length < 2) {
      setUserSuggestions([]);
      setIsUserSearchLoading(false);
      return;
    }

    let isActive = true;
    setIsUserSearchLoading(true);
    const timeoutId = window.setTimeout(() => {
      assetService
        .searchUsers(query)
        .then((results) => {
          if (!isActive) return;
          setUserSuggestions(results);
        })
        .catch((err) => {
          if (!isActive) return;
          console.error('Failed to search users:', err);
          setUserSuggestions([]);
        })
        .finally(() => {
          if (!isActive) return;
          setIsUserSearchLoading(false);
        });
    }, 300);

    return () => {
      isActive = false;
      clearTimeout(timeoutId);
    };
  }, [showAddUserModal, userFormData.user_id]);

  const loadRealm = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await assetService.getRealm(realmId!);
      setRealm(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load realm');
    } finally {
      setLoading(false);
    }
  };

  const loadUsersCount = async () => {
    try {
      const data = await assetService.getRealmUsers(realmId!, 1, 0);
      setUsersTotal(data.total);
    } catch (err) {
      console.error('Failed to load users count:', err);
    }
  };

  const handleSelectUserSuggestion = (suggestion: UserSuggestion) => {
    setUserFormData((prev) => ({ ...prev, user_id: suggestion.id }));
    setUserSuggestions([]);
  };

  const loadAssets = async () => {
    try {
      const params: any = {
        limit: assetsLimit,
        offset: assetsPage * assetsLimit,
      };
      if (assetTypeFilter) params.type = assetTypeFilter;
      if (assetSearch) params.search = assetSearch;

      const data = await assetService.getAssets(realmId!, params);
      setAssets(data.data);
      setAssetsTotal(data.total);
    } catch (err) {
      console.error('Failed to load assets:', err);
    }
  };

  const loadUsers = async () => {
    try {
      const data = await assetService.getRealmUsers(realmId!, usersLimit, usersPage * usersLimit);
      setUsers(data.data);
      setUsersTotal(data.total);
    } catch (err) {
      console.error('Failed to load users:', err);
    }
  };

  const handleDeleteRealm = async () => {
    if (!confirm('Are you sure you want to delete this realm? This action cannot be undone.')) {
      return;
    }

    try {
      await assetService.deleteRealm(realmId!);
      navigate('/realms');
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete realm');
    }
  };

  const loadAllRealmUsers = async () => {
    try {
      // Load all users (no pagination)
      const data = await assetService.getRealmUsers(realmId!, 1000, 0);
      setAllRealmUsers(data.data);
    } catch (err) {
      console.error('Failed to load realm users:', err);
    }
  };

  const handleCreateAsset = async (e: React.FormEvent) => {
    e.preventDefault();
    setAssetFormError('');

    try {
      const data = {
        ...assetFormData,
        owner_user_id: assetFormData.owner_user_id || user?.id // Default to current user if not selected
      };
      await assetService.createAsset(realmId!, data);
      setShowCreateAssetModal(false);
      setAssetFormData({ name: '', asset_type: 'spirits', description: '', owner_user_id: '' });
      await loadAssets();
    } catch (err) {
      setAssetFormError(err instanceof Error ? err.message : 'Failed to create asset');
    }
  };

  const handleAddUser = async (e: React.FormEvent) => {
    e.preventDefault();
    setUserFormError('');

    try {
      await assetService.addUserToRealm(realmId!, userFormData);
      setShowAddUserModal(false);
      setUserFormData({ user_id: '', role: 'member' });
      setUserSuggestions([]);
      await loadUsers();
    } catch (err) {
      setUserFormError(err instanceof Error ? err.message : 'Failed to add user');
    }
  };

  // User removal with asset handling
  const [showRemoveUserModal, setShowRemoveUserModal] = useState(false);
  const [userToRemove, setUserToRemove] = useState<{id: string; username: string; assets: Asset[]} | null>(null);
  const [removeUserAction, setRemoveUserAction] = useState<'reassign' | 'delete'>('reassign');
  const [newAssetOwner, setNewAssetOwner] = useState('');
  const [removeUserError, setRemoveUserError] = useState('');

  const handleRemoveUser = async (userId: string, username: string) => {
    try {
      // Try to remove user
      await assetService.removeUserFromRealm(realmId!, userId);
      await loadUsers();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to remove user';

      // Check if error is about assets ownership
      if (errorMessage.includes('assets')) {
        // Load user's assets
        try {
          const userAssets = await assetService.getUserAssets(realmId!, userId);
          setUserToRemove({ id: userId, username, assets: userAssets });
          setShowRemoveUserModal(true);
        } catch (assetsErr) {
          alert('Failed to load user assets');
        }
      } else {
        alert(errorMessage);
      }
    }
  };

  const handleConfirmRemoveUser = async () => {
    if (!userToRemove) return;

    setRemoveUserError('');

    try {
      if (removeUserAction === 'reassign') {
        if (!newAssetOwner) {
          setRemoveUserError('Please select a new owner');
          return;
        }
        await assetService.reassignUserAssets(realmId!, userToRemove.id, newAssetOwner);
      } else {
        await assetService.deleteUserAssets(realmId!, userToRemove.id);
      }

      // Now remove the user
      await assetService.removeUserFromRealm(realmId!, userToRemove.id);

      setShowRemoveUserModal(false);
      setUserToRemove(null);
      setNewAssetOwner('');
      await loadUsers();
      await loadAssets(); // Reload assets to reflect changes
    } catch (err) {
      setRemoveUserError(err instanceof Error ? err.message : 'Failed to remove user');
    }
  };

  const handleEditRealm = async (e: React.FormEvent) => {
    e.preventDefault();
    setRealmEditError('');

    try {
      const updatedRealm = await assetService.updateRealm(realmId!, realmEditData);
      // Preserve the role from the current realm state
      setRealm({ ...updatedRealm, role: realm?.role });
      setShowEditRealmModal(false);
    } catch (err) {
      setRealmEditError(err instanceof Error ? err.message : 'Failed to update realm');
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-orange-600"></div>
      </div>
    );
  }

  if (error || !realm) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <AlertCircle className="h-16 w-16 text-red-500 mx-auto mb-4" />
          <p className="text-xl text-gray-900 mb-2">Failed to load realm</p>
          <p className="text-gray-600 mb-4">{error}</p>
          <Link to="/realms" className="text-orange-600 hover:text-orange-700">
            Back to Bars
          </Link>
        </div>
      </div>
    );
  }

  const assetsTotalPages = Math.ceil(assetsTotal / assetsLimit);
  const usersTotalPages = Math.ceil(usersTotal / usersLimit);

  return (
    <div className="min-h-screen bg-gradient-to-br from-orange-50 to-amber-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-6">
          <Link
            to="/realms"
            className="inline-flex items-center gap-2 text-gray-600 hover:text-gray-900 mb-4"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to Bars
          </Link>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-start justify-between">
              <div>
                <div className="flex items-center gap-3 mb-2">
                  <h1 className="text-3xl font-bold text-gray-900">{realm.name}</h1>
                  <RoleBadge role={realm.role!} />
                </div>
                <p className="text-gray-600 mb-4">{realm.description || 'No description'}</p>
                <p className="text-sm text-gray-500">
                  Created {new Date(realm.created_at).toLocaleDateString()}
                </p>
              </div>
              {isAdmin && (
                <div className="flex gap-2">
                  <button
                    onClick={() => {
                      setRealmEditData({ name: realm.name, description: realm.description || '' });
                      setShowEditRealmModal(true);
                    }}
                    className="p-2 text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
                  >
                    <Edit className="h-5 w-5" />
                  </button>
                  <button
                    onClick={handleDeleteRealm}
                    className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                  >
                    <Trash2 className="h-5 w-5" />
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow">
          <div className="border-b border-gray-200">
            <div className="flex">
              <button
                onClick={() => setActiveTab('assets')}
                className={`flex items-center gap-2 px-6 py-4 font-medium border-b-2 transition-colors ${
                  activeTab === 'assets'
                    ? 'border-orange-600 text-orange-600'
                    : 'border-transparent text-gray-600 hover:text-gray-900'
                }`}
              >
                <Package className="h-5 w-5" />
                Assets ({assetsTotal})
              </button>
              <button
                onClick={() => setActiveTab('users')}
                className={`flex items-center gap-2 px-6 py-4 font-medium border-b-2 transition-colors ${
                  activeTab === 'users'
                    ? 'border-orange-600 text-orange-600'
                    : 'border-transparent text-gray-600 hover:text-gray-900'
                }`}
              >
                <Users className="h-5 w-5" />
                Users ({usersTotal})
              </button>
            </div>
          </div>

          {activeTab === 'assets' && (
            <div>
              <div className="p-6 border-b border-gray-200">
                <div className="flex flex-wrap items-center gap-4">
                  <div className="flex-1 min-w-[200px]">
                    <div className="relative">
                      <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-gray-400" />
                      <input
                        type="text"
                        value={assetSearch}
                        onChange={(e) => {
                          setAssetSearch(e.target.value);
                          setAssetsPage(0);
                        }}
                        placeholder="Search assets..."
                        className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                      />
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <Filter className="h-5 w-5 text-gray-600" />
                    <select
                      value={assetTypeFilter}
                      onChange={(e) => {
                        setAssetTypeFilter(e.target.value as AssetType | '');
                        setAssetsPage(0);
                      }}
                      className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    >
                      <option value="">All Types</option>
                      <option value="server">Server</option>
                      <option value="application">Application</option>
                      <option value="database">Database</option>
                      <option value="network">Network</option>
                      <option value="other">Other</option>
                    </select>
                  </div>

                  <button
                    onClick={() => setShowCreateAssetModal(true)}
                    className="inline-flex items-center gap-2 px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors"
                  >
                    <Plus className="h-4 w-4" />
                    Create Asset
                  </button>
                </div>
              </div>

              <div className="p-6">
                {assets.length === 0 ? (
                  <div className="text-center py-12">
                    <Package className="h-16 w-16 text-gray-400 mx-auto mb-4" />
                    <p className="text-gray-600 mb-4">No assets found</p>
                    <button
                      onClick={() => setShowCreateAssetModal(true)}
                      className="inline-flex items-center gap-2 px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors"
                    >
                      <Plus className="h-4 w-4" />
                      Create First Asset
                    </button>
                  </div>
                ) : (
                  <>
                    <div className="grid gap-4 mb-6">
                      {assets.map((asset) => (
                        <Link
                          key={asset.id}
                          to={`/assets/${asset.id}`}
                          className="flex items-center justify-between p-4 border border-gray-200 rounded-lg hover:border-orange-300 hover:bg-orange-50 transition-all"
                        >
                          <div className="flex items-center gap-4">
                            <div className="p-2 bg-orange-100 rounded-lg">
                              <Package className="h-6 w-6 text-orange-600" />
                            </div>
                            <div>
                              <div className="flex items-center gap-2 mb-1">
                                <h3 className="font-semibold text-gray-900">{asset.name}</h3>
                                <AssetTypeBadge type={asset.asset_type} />
                              </div>
                              <p className="text-sm text-gray-600">
                                {asset.description || 'No description'}
                              </p>
                            </div>
                          </div>
                          <p className="text-xs text-gray-500">
                            {new Date(asset.created_at).toLocaleDateString()}
                          </p>
                        </Link>
                      ))}
                    </div>

                    {assetsTotalPages > 1 && (
                      <div className="flex items-center justify-between">
                        <p className="text-sm text-gray-600">
                          Showing {assetsPage * assetsLimit + 1} to{' '}
                          {Math.min((assetsPage + 1) * assetsLimit, assetsTotal)} of {assetsTotal}
                        </p>
                        <div className="flex gap-2">
                          <button
                            onClick={() => setAssetsPage((p) => Math.max(0, p - 1))}
                            disabled={assetsPage === 0}
                            className="p-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            <ChevronLeft className="h-5 w-5" />
                          </button>
                          <button
                            onClick={() => setAssetsPage((p) => Math.min(assetsTotalPages - 1, p + 1))}
                            disabled={assetsPage >= assetsTotalPages - 1}
                            className="p-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            <ChevronRight className="h-5 w-5" />
                          </button>
                        </div>
                      </div>
                    )}
                  </>
                )}
              </div>
            </div>
          )}

          {activeTab === 'users' && (
            <div>
              <div className="p-6 border-b border-gray-200">
                <div className="flex items-center justify-between">
                  <h3 className="text-lg font-semibold text-gray-900">Bar Staff</h3>
                  {isAdmin && (
                    <button
                      onClick={() => setShowAddUserModal(true)}
                      className="inline-flex items-center gap-2 px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors"
                    >
                      <Plus className="h-4 w-4" />
                      Add User
                    </button>
                  )}
                </div>
              </div>

              <div className="p-6">
                {users.length === 0 ? (
                  <div className="text-center py-12">
                    <Users className="h-16 w-16 text-gray-400 mx-auto mb-4" />
                    <p className="text-gray-600">No staff members in this bar</p>
                  </div>
                ) : (
                  <>
                    <div className="space-y-4 mb-6">
                      {users.map((realmUser) => (
                        <div
                          key={realmUser.user_id}
                          className="flex items-center justify-between p-4 border border-gray-200 rounded-lg"
                        >
                      <div>
                        <p className="font-medium text-gray-900">
                          {realmUser.username || realmUser.user_id}
                        </p>
                        {realmUser.username && (
                          <p className="text-sm text-gray-600">{realmUser.user_id}</p>
                        )}
                        <p className="text-sm text-gray-600">
                          Added {new Date(realmUser.added_at).toLocaleDateString()}
                        </p>
                      </div>
                          <div className="flex items-center gap-3">
                            <RoleBadge role={realmUser.role} />
                            {isAdmin && realmUser.user_id !== user?.id && (
                              <button
                                onClick={() => handleRemoveUser(realmUser.user_id, realmUser.username || realmUser.user_id)}
                                className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                              >
                                <Trash2 className="h-4 w-4" />
                              </button>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>

                    {usersTotalPages > 1 && (
                      <div className="flex items-center justify-between">
                        <p className="text-sm text-gray-600">
                          Showing {usersPage * usersLimit + 1} to{' '}
                          {Math.min((usersPage + 1) * usersLimit, usersTotal)} of {usersTotal}
                        </p>
                        <div className="flex gap-2">
                          <button
                            onClick={() => setUsersPage((p) => Math.max(0, p - 1))}
                            disabled={usersPage === 0}
                            className="p-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            <ChevronLeft className="h-5 w-5" />
                          </button>
                          <button
                            onClick={() => setUsersPage((p) => Math.min(usersTotalPages - 1, p + 1))}
                            disabled={usersPage >= usersTotalPages - 1}
                            className="p-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            <ChevronRight className="h-5 w-5" />
                          </button>
                        </div>
                      </div>
                    )}
                  </>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Create Asset Modal */}
        {showCreateAssetModal && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg shadow-xl max-w-lg w-full p-6 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-2xl font-bold text-gray-900">Add Inventory Item</h2>
                <button
                  onClick={() => {
                    setShowCreateAssetModal(false);
                    setAssetFormData({ name: '', asset_type: 'spirits', description: '', owner_user_id: '' });
                    setAssetFormError('');
                  }}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <X className="h-6 w-6" />
                </button>
              </div>

              {assetFormError && (
                <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-center gap-2 text-red-800 text-sm">
                  <AlertCircle className="h-4 w-4" />
                  <span>{assetFormError}</span>
                </div>
              )}

              <form onSubmit={handleCreateAsset} className="space-y-4">
                <div>
                  <label htmlFor="asset-name" className="block text-sm font-medium text-gray-700 mb-1">
                    Item Name *
                  </label>
                  <input
                    id="asset-name"
                    type="text"
                    value={assetFormData.name}
                    onChange={(e) => setAssetFormData({ ...assetFormData, name: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    placeholder="e.g., Jack Daniel's, Corona Extra, Lime wedges"
                    required
                  />
                </div>

                <div>
                  <label htmlFor="asset-type" className="block text-sm font-medium text-gray-700 mb-1">
                    Category *
                  </label>
                  <select
                    id="asset-type"
                    value={assetFormData.asset_type}
                    onChange={(e) => setAssetFormData({ ...assetFormData, asset_type: e.target.value as AssetType })}
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
                    value={assetFormData.description}
                    onChange={(e) => setAssetFormData({ ...assetFormData, description: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    rows={3}
                    placeholder="Additional details about this item..."
                  />
                </div>

                <div>
                  <label htmlFor="asset-owner" className="block text-sm font-medium text-gray-700 mb-1">
                    Owner
                  </label>
                  <select
                    id="asset-owner"
                    value={assetFormData.owner_user_id}
                    onChange={(e) => setAssetFormData({ ...assetFormData, owner_user_id: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                  >
                    <option value="">Current User (default)</option>
                    {allRealmUsers.map((realmUser) => (
                      <option key={realmUser.user_id} value={realmUser.user_id}>
                        {realmUser.username ? `${realmUser.username} (${realmUser.user_id})` : realmUser.user_id}
                      </option>
                    ))}
                  </select>
                  <p className="mt-1 text-xs text-gray-500">
                    Leave as default to assign to yourself
                  </p>
                </div>

                <div className="flex gap-3">
                  <button
                    type="submit"
                    className="flex-1 py-2 bg-orange-600 text-white rounded-lg font-semibold hover:bg-orange-700 transition-colors"
                  >
                    Add Item
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setShowCreateAssetModal(false);
                      setAssetFormData({ name: '', asset_type: 'spirits', description: '', owner_user_id: '' });
                      setAssetFormError('');
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

        {/* Add User Modal */}
        {showAddUserModal && isAdmin && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg shadow-xl max-w-md w-full p-6">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-2xl font-bold text-gray-900">Add Staff Bartender</h2>
                <button
                  onClick={() => {
                    setShowAddUserModal(false);
                    setUserFormData({ user_id: '', role: 'member' });
                    setUserFormError('');
                    setUserSuggestions([]);
                  }}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <X className="h-6 w-6" />
                </button>
              </div>

              {userFormError && (
                <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-center gap-2 text-red-800 text-sm">
                  <AlertCircle className="h-4 w-4" />
                  <span>{userFormError}</span>
                </div>
              )}

              <form onSubmit={handleAddUser} className="space-y-4">
                <div className="relative">
                  <label htmlFor="user-id" className="block text-sm font-medium text-gray-700 mb-1">
                    User ID or Username *
                  </label>
                  <input
                    id="user-id"
                    type="text"
                    value={userFormData.user_id}
                    onChange={(e) => setUserFormData({ ...userFormData, user_id: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    placeholder="username or user UUID"
                    autoComplete="off"
                    required
                  />
                  {showAddUserModal && userFormData.user_id.trim().length >= 2 && (
                    <div className="absolute left-0 right-0 mt-1 bg-white border border-gray-200 rounded-lg shadow-lg z-10 max-h-60 overflow-y-auto">
                      {isUserSearchLoading ? (
                        <div className="px-4 py-3 text-sm text-gray-500">Searching...</div>
                      ) : userSuggestions.length === 0 ? (
                        <div className="px-4 py-3 text-sm text-gray-500">No matching users</div>
                      ) : (
                        userSuggestions.map((suggestion) => (
                          <button
                            key={suggestion.id}
                            type="button"
                            onClick={() => handleSelectUserSuggestion(suggestion)}
                            className="w-full text-left px-4 py-2 hover:bg-orange-50 focus:outline-none"
                          >
                            <div className="text-sm font-medium text-gray-900">{suggestion.username}</div>
                            <div className="text-xs text-gray-500 truncate">{suggestion.id}</div>
                            <div className="text-xs text-gray-500 truncate">{suggestion.email}</div>
                          </button>
                        ))
                      )}
                    </div>
                  )}
                </div>

                <div>
                  <label htmlFor="user-role" className="block text-sm font-medium text-gray-700 mb-1">
                    Role *
                  </label>
                  <select
                    id="user-role"
                    value={userFormData.role}
                    onChange={(e) => setUserFormData({ ...userFormData, role: e.target.value as 'admin' | 'member' })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    required
                  >
                    <option value="member">Bartender</option>
                    <option value="admin">Bar Manager</option>
                  </select>
                </div>

                <div className="flex gap-3">
                  <button
                    type="submit"
                    className="flex-1 py-2 bg-orange-600 text-white rounded-lg font-semibold hover:bg-orange-700 transition-colors"
                  >
                    Add User
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setShowAddUserModal(false);
                      setUserFormData({ user_id: '', role: 'member' });
                      setUserFormError('');
                      setUserSuggestions([]);
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

        {/* Edit Bar Modal */}
        {showEditRealmModal && isAdmin && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg shadow-xl max-w-md w-full p-6">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-2xl font-bold text-gray-900">Edit Bar</h2>
                <button
                  onClick={() => {
                    setShowEditRealmModal(false);
                    setRealmEditError('');
                  }}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <X className="h-6 w-6" />
                </button>
              </div>

              {realmEditError && (
                <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-center gap-2 text-red-800 text-sm">
                  <AlertCircle className="h-4 w-4" />
                  <span>{realmEditError}</span>
                </div>
              )}

              <form onSubmit={handleEditRealm} className="space-y-4">
                <div>
                  <label htmlFor="realm-name" className="block text-sm font-medium text-gray-700 mb-1">
                    Realm Name *
                  </label>
                  <input
                    id="realm-name"
                    type="text"
                    value={realmEditData.name}
                    onChange={(e) => setRealmEditData({ ...realmEditData, name: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    placeholder="My Bar"
                    required
                  />
                </div>

                <div>
                  <label htmlFor="realm-description" className="block text-sm font-medium text-gray-700 mb-1">
                    Description
                  </label>
                  <textarea
                    id="realm-description"
                    value={realmEditData.description}
                    onChange={(e) => setRealmEditData({ ...realmEditData, description: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    rows={3}
                    placeholder="Description of the realm..."
                  />
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
                      setShowEditRealmModal(false);
                      setRealmEditError('');
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

        {/* Remove User with Assets Modal */}
        {showRemoveUserModal && userToRemove && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg shadow-xl max-w-lg w-full p-6 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-2xl font-bold text-gray-900">Remove User with Assets</h2>
                <button
                  onClick={() => {
                    setShowRemoveUserModal(false);
                    setUserToRemove(null);
                    setRemoveUserError('');
                    setNewAssetOwner('');
                  }}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <X className="h-6 w-6" />
                </button>
              </div>

              {removeUserError && (
                <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-center gap-2 text-red-800 text-sm">
                  <AlertCircle className="h-4 w-4" />
                  <span>{removeUserError}</span>
                </div>
              )}

              <div className="mb-6">
                <p className="text-gray-900 mb-4">
                  User <strong>{userToRemove.username}</strong> owns <strong>{userToRemove.assets.length}</strong> asset(s) in this realm:
                </p>
                <div className="bg-gray-50 rounded-lg p-4 border border-gray-200 max-h-40 overflow-y-auto">
                  <ul className="space-y-2">
                    {userToRemove.assets.map((asset) => (
                      <li key={asset.id} className="text-sm text-gray-700">
                        <strong>{asset.name}</strong> ({asset.asset_type})
                      </li>
                    ))}
                  </ul>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    What should happen to these assets?
                  </label>
                  <div className="space-y-2">
                    <label className="flex items-center gap-2 p-3 border border-gray-300 rounded-lg cursor-pointer hover:bg-gray-50">
                      <input
                        type="radio"
                        name="action"
                        value="reassign"
                        checked={removeUserAction === 'reassign'}
                        onChange={() => setRemoveUserAction('reassign')}
                        className="text-orange-600 focus:ring-orange-500"
                      />
                      <div>
                        <div className="font-medium text-gray-900">Reassign to another user</div>
                        <div className="text-sm text-gray-600">Transfer ownership to another realm member</div>
                      </div>
                    </label>
                    <label className="flex items-center gap-2 p-3 border border-gray-300 rounded-lg cursor-pointer hover:bg-gray-50">
                      <input
                        type="radio"
                        name="action"
                        value="delete"
                        checked={removeUserAction === 'delete'}
                        onChange={() => setRemoveUserAction('delete')}
                        className="text-orange-600 focus:ring-orange-500"
                      />
                      <div>
                        <div className="font-medium text-gray-900">Delete all assets</div>
                        <div className="text-sm text-gray-600 text-red-600">This action cannot be undone</div>
                      </div>
                    </label>
                  </div>
                </div>

                {removeUserAction === 'reassign' && (
                  <div>
                    <label htmlFor="new-owner" className="block text-sm font-medium text-gray-700 mb-1">
                      New Owner *
                    </label>
                    <select
                      id="new-owner"
                      value={newAssetOwner}
                      onChange={(e) => setNewAssetOwner(e.target.value)}
                      className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                      required
                    >
                      <option value="">Select new owner...</option>
                      {users
                        .filter((u) => u.user_id !== userToRemove.id)
                        .map((realmUser) => (
                          <option key={realmUser.user_id} value={realmUser.user_id}>
                            {realmUser.username ? `${realmUser.username} (${realmUser.user_id})` : realmUser.user_id}
                          </option>
                        ))}
                    </select>
                  </div>
                )}

                <div className="flex gap-3 pt-4">
                  <button
                    onClick={handleConfirmRemoveUser}
                    className="flex-1 py-2 bg-red-600 text-white rounded-lg font-semibold hover:bg-red-700 transition-colors"
                  >
                    {removeUserAction === 'reassign' ? 'Reassign & Remove User' : 'Delete Assets & Remove User'}
                  </button>
                  <button
                    onClick={() => {
                      setShowRemoveUserModal(false);
                      setUserToRemove(null);
                      setRemoveUserError('');
                      setNewAssetOwner('');
                    }}
                    className="flex-1 py-2 bg-gray-200 text-gray-700 rounded-lg font-semibold hover:bg-gray-300 transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
