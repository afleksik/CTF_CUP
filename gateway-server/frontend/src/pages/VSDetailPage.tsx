import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { gatewayService } from '../services/gatewayService';
import type { VSWithFeeds, UpdateVSData, TIMode, TrafficLog, VirtualServiceUser } from '../types';
import {
  ArrowLeft,
  AlertCircle,
  Settings,
  Power,
  Trash2,
  Shield,
  Plus,
  X,
  Activity,
  Users,
  Copy,
  Check,
  RefreshCw,
  ExternalLink
} from 'lucide-react';

export default function VSDetailPage() {
  const { vsId } = useParams<{ vsId: string }>();
  const { user } = useAuth();
  const navigate = useNavigate();

  const [service, setService] = useState<VSWithFeeds | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Edit modal
  const [showEditModal, setShowEditModal] = useState(false);
  const [editFormData, setEditFormData] = useState<UpdateVSData>({});
  const [editError, setEditError] = useState('');

  // TI Feeds
  const [showAddFeedModal, setShowAddFeedModal] = useState(false);
  const [feedId, setFeedId] = useState('');
  const [feedApiKey, setFeedApiKey] = useState('');

  // Traffic logs
  const [logs, setLogs] = useState<TrafficLog[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [showBlockedOnly, setShowBlockedOnly] = useState(false);

  // Service users
  const [showAddUserModal, setShowAddUserModal] = useState(false);
  const [serviceUsers, setServiceUsers] = useState<VirtualServiceUser[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [newUserId, setNewUserId] = useState('');

  // Copy state
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!user) {
      navigate('/');
      return;
    }
    if (vsId) {
      loadService();
      loadLogs();
    }
  }, [vsId, user, navigate]);

  useEffect(() => {
    if (service?.require_auth) {
      loadUsers();
    }
  }, [service?.require_auth]);

  useEffect(() => {
    if (vsId) {
      loadLogs();
    }
  }, [showBlockedOnly]);

  const loadService = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await gatewayService.getVirtualService(vsId!);
      setService(data);
      setEditFormData({
        name: data.name,
        backend_url: data.backend_url,
        is_active: data.is_active,
        require_auth: data.require_auth,
        ti_mode: data.ti_mode,
        rate_limit_enabled: data.rate_limit_enabled,
        rate_limit_requests: data.rate_limit_requests,
        rate_limit_window_sec: data.rate_limit_window_sec,
        log_retention_minutes: data.log_retention_minutes,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load virtual service');
    } finally {
      setLoading(false);
    }
  };

  const loadLogs = async () => {
    try {
      setLogsLoading(true);
      const response = await gatewayService.getTrafficLogs(vsId!, { limit: 20, blocked: showBlockedOnly || undefined });
      setLogs(response.data);
    } catch (err) {
      console.error('Failed to load logs:', err);
    } finally {
      setLogsLoading(false);
    }
  };

  const loadUsers = async () => {
    try {
      setUsersLoading(true);
      const users = await gatewayService.getVSUsers(vsId!);
      setServiceUsers(users);
    } catch (err) {
      console.error('Failed to load users:', err);
    } finally {
      setUsersLoading(false);
    }
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    setEditError('');
    try {
      await gatewayService.updateVirtualService(vsId!, editFormData);
      await loadService();
      setShowEditModal(false);
    } catch (err) {
      setEditError(err instanceof Error ? err.message : 'Failed to update service');
    }
  };

  const handleToggleActive = async () => {
    try {
      await gatewayService.updateVirtualService(vsId!, { is_active: !service?.is_active });
      await loadService();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to toggle service');
    }
  };

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this service?')) return;
    try {
      await gatewayService.deleteVirtualService(vsId!);
      navigate('/services');
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete service');
    }
  };

  const handleLoadAvailableFeeds = async () => {
    setShowAddFeedModal(true);
  };

  const handleAttachFeed = async () => {
    if (!feedId.trim()) {
      alert('Please enter feed ID');
      return;
    }

    try {
      const attachData: any = { feed_id: feedId.trim() };

      // If there's an API key entered, include it
      if (feedApiKey.trim()) {
        attachData.api_key = feedApiKey.trim();
      }

      await gatewayService.attachTIFeed(vsId!, attachData);
      await loadService();
      setShowAddFeedModal(false);
      setFeedId('');
      setFeedApiKey('');
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to attach feed');
    }
  };

  const handleToggleFeed = async (feedId: string, isActive: boolean) => {
    try {
      await gatewayService.toggleTIFeed(vsId!, feedId, !isActive);
      await loadService();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to toggle feed');
    }
  };

  const handleDetachFeed = async (feedId: string) => {
    if (!confirm('Detach this TI feed?')) return;
    try {
      await gatewayService.detachTIFeed(vsId!, feedId);
      await loadService();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to detach feed');
    }
  };

  const handleAddUser = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await gatewayService.addUserToVS(vsId!, { user_id: newUserId });
      setNewUserId('');
      setShowAddUserModal(false);
      await loadUsers();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to add user');
    }
  };

  const handleRemoveUser = async (userId: string) => {
    if (!confirm('Remove this user from service?')) return;
    try {
      await gatewayService.removeUserFromVS(vsId!, userId);
      await loadUsers();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to remove user');
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const gatewayUrl = `${window.location.protocol}//${window.location.host}/vs/${service?.slug}`;

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-purple-50 via-violet-50 to-indigo-50">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600"></div>
      </div>
    );
  }

  if (error || !service) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-purple-50 via-violet-50 to-indigo-50">
        <div className="text-center">
          <AlertCircle className="h-16 w-16 text-red-500 mx-auto mb-4" />
          <p className="text-xl text-gray-900 mb-2">Failed to load service</p>
          <p className="text-gray-600 mb-4">{error}</p>
          <Link to="/services" className="text-purple-600 hover:text-purple-700 font-medium">
            Back to Services
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-purple-50 via-violet-50 to-indigo-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Link
          to="/services"
          className="inline-flex items-center gap-2 text-gray-600 hover:text-gray-900 mb-6"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Services
        </Link>

        {/* Header Card */}
        <div className="bg-white rounded-xl shadow-lg p-6 border border-purple-100 mb-6">
          <div className="flex items-start justify-between mb-6">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-gradient-to-br from-purple-500 to-violet-600 rounded-xl">
                <svg className="h-8 w-8" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <circle cx="12" cy="12" r="3.5" fill="white"/>
                  <circle cx="12" cy="12" r="7" stroke="white" strokeWidth="1.5" fill="none" opacity="0.3"/>
                  <path d="M2 12 L7 12 M4 9.5 L2 12 L4 14.5" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
                  <path d="M12 2 L12 7 M9.5 4 L12 2 L14.5 4" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
                  <path d="M17 12 L22 12 M20 9.5 L22 12 L20 14.5" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
                  <path d="M12 17 L12 22 M9.5 20 L12 22 L14.5 20" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
              </div>
              <div>
                <div className="flex items-center gap-3">
                  <h1 className="text-3xl font-bold text-gray-900">{service.name}</h1>
                  <span
                    className={`px-3 py-1 text-sm font-medium rounded-lg ${
                      service.is_active
                        ? 'bg-green-100 text-green-800'
                        : 'bg-gray-100 text-gray-800'
                    }`}
                  >
                    {service.is_active ? 'Active' : 'Inactive'}
                  </span>
                </div>
                <p className="text-gray-600 mt-1 font-mono">/{service.slug}</p>
              </div>
            </div>

            <div className="flex gap-2">
              <button
                onClick={handleToggleActive}
                className={`p-2 rounded-lg border transition-all ${
                  service.is_active
                    ? 'border-gray-300 hover:bg-gray-50 text-gray-700'
                    : 'border-green-300 hover:bg-green-50 text-green-700'
                }`}
                title={service.is_active ? 'Deactivate' : 'Activate'}
              >
                <Power className="h-5 w-5" />
              </button>
              <button
                onClick={() => setShowEditModal(true)}
                className="p-2 rounded-lg border border-gray-300 hover:bg-gray-50 text-gray-700 transition-all"
                title="Edit"
              >
                <Settings className="h-5 w-5" />
              </button>
              <button
                onClick={handleDelete}
                className="p-2 rounded-lg border border-red-300 hover:bg-red-50 text-red-700 transition-all"
                title="Delete"
              >
                <Trash2 className="h-5 w-5" />
              </button>
            </div>
          </div>

          {/* Service Info Grid */}
          <div className="grid md:grid-cols-3 gap-6">
            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">Backend URL</h3>
              <p className="text-gray-900 font-mono text-sm">{service.backend_url}</p>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">TI Mode</h3>
              <p className="text-gray-900 capitalize">{service.ti_mode}</p>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">Authentication</h3>
              <p className="text-gray-900">{service.require_auth ? 'Required' : 'Not Required'}</p>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">Rate Limiting</h3>
              <p className="text-gray-900">
                {service.rate_limit_enabled
                  ? `${service.rate_limit_requests} req / ${service.rate_limit_window_sec}s`
                  : 'Disabled'}
              </p>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">Log Retention</h3>
              <p className="text-gray-900">{service.log_retention_minutes} minutes</p>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">Created</h3>
              <p className="text-gray-900 text-sm">{new Date(service.created_at).toLocaleString()}</p>
            </div>
          </div>
        </div>

        {/* Testing Information */}
        <div className="bg-white rounded-xl shadow-lg p-6 border border-purple-100 mb-6">
          <h2 className="text-xl font-semibold text-gray-900 mb-4 flex items-center gap-2">
            <ExternalLink className="h-5 w-5 text-purple-600" />
            Testing Information
          </h2>
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium text-gray-700 mb-2 block">Gateway URL</label>
              <div className="flex items-center gap-2">
                <code className="flex-1 px-4 py-2 bg-purple-50 border border-purple-200 rounded-lg text-purple-900 font-mono text-sm">
                  {gatewayUrl}/*
                </code>
                <button
                  onClick={() => copyToClipboard(gatewayUrl)}
                  className="p-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-all"
                  title="Copy URL"
                >
                  {copied ? <Check className="h-5 w-5 text-green-600" /> : <Copy className="h-5 w-5 text-gray-600" />}
                </button>
              </div>
            </div>

            <div>
              <label className="text-sm font-medium text-gray-700 mb-2 block">Example curl command</label>
              <code className="block px-4 py-3 bg-gray-900 text-gray-100 rounded-lg text-sm font-mono overflow-x-auto">
                curl {service.require_auth ? '-H "Authorization: Bearer YOUR_TOKEN" ' : ''}{gatewayUrl}/path
              </code>
            </div>

            {service.require_auth && (
              <div className="p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
                <p className="text-sm text-yellow-900">
                  <strong>Note:</strong> This service requires authentication. Make sure to include a valid JWT token in the Authorization header.
                </p>
              </div>
            )}
          </div>
        </div>

        {/* TI Feeds Section */}
        <div className="bg-white rounded-xl shadow-lg p-6 border border-purple-100 mb-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-semibold text-gray-900 flex items-center gap-2">
              <Shield className="h-5 w-5 text-purple-600" />
              Threat Intelligence Feeds
            </h2>
            <button
              onClick={handleLoadAvailableFeeds}
              className="inline-flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-purple-600 to-violet-600 text-white rounded-lg hover:from-purple-700 hover:to-violet-700 transition-all shadow-md hover:shadow-lg"
            >
              <Plus className="h-4 w-4" />
              Add Feed
            </button>
          </div>

          {service.ti_feeds && service.ti_feeds.length > 0 ? (
            <div className="space-y-2">
              {service.ti_feeds.map((feed) => (
                <div
                  key={feed.feed_id}
                  className="flex items-center justify-between p-4 bg-purple-50 border border-purple-100 rounded-lg"
                >
                  <div>
                    <p className="font-medium text-gray-900">{feed.feed_name}</p>
                    <p className="text-sm text-gray-600">ID: {feed.feed_id}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <span
                      className={`px-3 py-1 text-sm font-medium rounded-lg ${
                        feed.is_active
                          ? 'bg-green-100 text-green-800'
                          : 'bg-gray-100 text-gray-800'
                      }`}
                    >
                      {feed.is_active ? 'Active' : 'Inactive'}
                    </span>
                    <button
                      onClick={() => handleToggleFeed(feed.feed_id, feed.is_active)}
                      className="p-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-all"
                      title={feed.is_active ? 'Deactivate' : 'Activate'}
                    >
                      <Power className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => handleDetachFeed(feed.feed_id)}
                      className="p-2 border border-red-300 rounded-lg hover:bg-red-50 text-red-700 transition-all"
                      title="Detach"
                    >
                      <X className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-gray-500">
              <Shield className="h-12 w-12 mx-auto mb-3 opacity-20" />
              <p>No TI feeds attached</p>
            </div>
          )}
        </div>

        {/* Service Users (if auth required) */}
        {service.require_auth && (
          <div className="bg-white rounded-xl shadow-lg p-6 border border-purple-100 mb-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold text-gray-900 flex items-center gap-2">
                <Users className="h-5 w-5 text-purple-600" />
                Authorized Users
              </h2>
              <button
                onClick={() => setShowAddUserModal(true)}
                className="inline-flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-purple-600 to-violet-600 text-white rounded-lg hover:from-purple-700 hover:to-violet-700 transition-all shadow-md hover:shadow-lg"
              >
                <Plus className="h-4 w-4" />
                Add User
              </button>
            </div>

            {usersLoading ? (
              <div className="text-center py-8">
                <RefreshCw className="h-8 w-8 animate-spin mx-auto text-purple-600" />
              </div>
            ) : serviceUsers.length > 0 ? (
              <div className="space-y-2">
                {serviceUsers.map((vsUser) => (
                  <div
                    key={vsUser.user_id}
                    className="flex items-center justify-between p-4 bg-purple-50 border border-purple-100 rounded-lg"
                  >
                    <div>
                      <p className="font-medium text-gray-900">{vsUser.username || vsUser.user_id}</p>
                      <p className="text-sm text-gray-600">Added {new Date(vsUser.granted_at).toLocaleDateString()}</p>
                    </div>
                    <button
                      onClick={() => handleRemoveUser(vsUser.user_id)}
                      className="p-2 border border-red-300 rounded-lg hover:bg-red-50 text-red-700 transition-all"
                      title="Remove"
                    >
                      <X className="h-4 w-4" />
                    </button>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8 text-gray-500">
                <Users className="h-12 w-12 mx-auto mb-3 opacity-20" />
                <p>No users authorized</p>
              </div>
            )}
          </div>
        )}

        {/* Traffic Logs */}
        <div className="bg-white rounded-xl shadow-lg p-6 border border-purple-100">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-semibold text-gray-900 flex items-center gap-2">
              <Activity className="h-5 w-5 text-purple-600" />
              Traffic Logs
            </h2>
            <div className="flex items-center gap-4">
              <label className="flex items-center gap-2 text-sm text-gray-700">
                <input
                  type="checkbox"
                  checked={showBlockedOnly}
                  onChange={(e) => setShowBlockedOnly(e.target.checked)}
                  className="w-4 h-4 text-purple-600 border-gray-300 rounded focus:ring-purple-500"
                />
                Blocked only
              </label>
              <button
                onClick={loadLogs}
                className="p-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-all"
                title="Refresh"
              >
                <RefreshCw className="h-4 w-4" />
              </button>
            </div>
          </div>

          {logsLoading ? (
            <div className="text-center py-8">
              <RefreshCw className="h-8 w-8 animate-spin mx-auto text-purple-600" />
            </div>
          ) : logs.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-purple-50 border-b border-purple-100">
                  <tr>
                    <th className="px-4 py-3 text-left font-medium text-gray-700">Time</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-700">Method</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-700">Path</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-700">Client IP</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-700">Status</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-700">IOC Matches</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-700">Blocked</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {logs.map((log) => (
                    <tr key={log.id} className={log.blocked ? 'bg-red-50' : ''}>
                      <td className="px-4 py-3 text-gray-900 whitespace-nowrap">
                        {new Date(log.timestamp).toLocaleTimeString()}
                      </td>
                      <td className="px-4 py-3 text-gray-900 font-mono">{log.method}</td>
                      <td className="px-4 py-3 text-gray-900 font-mono">{log.path}</td>
                      <td className="px-4 py-3 text-gray-900 font-mono">{log.client_ip}</td>
                      <td className="px-4 py-3">
                        <span className={`px-2 py-1 text-xs font-medium rounded ${
                          log.status_code < 400 ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                        }`}>
                          {log.status_code}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-gray-900">{log.ioc_matches?.length || 0}</td>
                      <td className="px-4 py-3">
                        {log.blocked ? (
                          <span className="px-2 py-1 text-xs font-medium rounded bg-red-100 text-red-800">Yes</span>
                        ) : (
                          <span className="px-2 py-1 text-xs font-medium rounded bg-green-100 text-green-800">No</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="text-center py-8 text-gray-500">
              <Activity className="h-12 w-12 mx-auto mb-3 opacity-20" />
              <p>No traffic logs yet</p>
            </div>
          )}
        </div>
      </div>

      {/* Edit Modal */}
      {showEditModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b border-gray-200">
              <div className="flex items-center justify-between">
                <h3 className="text-2xl font-bold text-gray-900">Edit Service</h3>
                <button
                  onClick={() => setShowEditModal(false)}
                  className="p-2 hover:bg-gray-100 rounded-lg transition-all"
                >
                  <X className="h-5 w-5" />
                </button>
              </div>
            </div>

            <form onSubmit={handleEdit} className="p-6 space-y-4">
              {editError && (
                <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-800 text-sm">
                  {editError}
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Service Name</label>
                <input
                  type="text"
                  value={editFormData.name || ''}
                  onChange={(e) => setEditFormData({ ...editFormData, name: e.target.value })}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Backend URL</label>
                <input
                  type="url"
                  value={editFormData.backend_url || ''}
                  onChange={(e) => setEditFormData({ ...editFormData, backend_url: e.target.value })}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">TI Mode</label>
                <select
                  value={editFormData.ti_mode || 'disabled'}
                  onChange={(e) => setEditFormData({ ...editFormData, ti_mode: e.target.value as TIMode })}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                >
                  <option value="disabled">Disabled</option>
                  <option value="monitor">Monitor</option>
                  <option value="block">Block</option>
                </select>
              </div>

              <div>
                <label className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    checked={editFormData.require_auth || false}
                    onChange={(e) => setEditFormData({ ...editFormData, require_auth: e.target.checked })}
                    className="w-4 h-4 text-purple-600 border-gray-300 rounded focus:ring-purple-500"
                  />
                  <span className="text-sm font-medium text-gray-700">Require Authentication</span>
                </label>
              </div>

              <div>
                <label className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    checked={editFormData.rate_limit_enabled || false}
                    onChange={(e) => setEditFormData({ ...editFormData, rate_limit_enabled: e.target.checked })}
                    className="w-4 h-4 text-purple-600 border-gray-300 rounded focus:ring-purple-500"
                  />
                  <span className="text-sm font-medium text-gray-700">Enable Rate Limiting</span>
                </label>
              </div>

              {editFormData.rate_limit_enabled && (
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Max Requests</label>
                    <input
                      type="number"
                      value={editFormData.rate_limit_requests || 100}
                      onChange={(e) => setEditFormData({ ...editFormData, rate_limit_requests: parseInt(e.target.value) })}
                      className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Window (seconds)</label>
                    <input
                      type="number"
                      value={editFormData.rate_limit_window_sec || 60}
                      onChange={(e) => setEditFormData({ ...editFormData, rate_limit_window_sec: parseInt(e.target.value) })}
                      className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                    />
                  </div>
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Log Retention (minutes)</label>
                <input
                  type="number"
                  value={editFormData.log_retention_minutes || 60}
                  onChange={(e) => setEditFormData({ ...editFormData, log_retention_minutes: parseInt(e.target.value) })}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                />
              </div>

              <div className="flex gap-4 pt-4">
                <button
                  type="submit"
                  className="px-6 py-2 bg-gradient-to-r from-purple-600 to-violet-600 text-white rounded-lg hover:from-purple-700 hover:to-violet-700 transition-all shadow-md hover:shadow-lg font-semibold"
                >
                  Save Changes
                </button>
                <button
                  type="button"
                  onClick={() => setShowEditModal(false)}
                  className="px-6 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-all font-semibold"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Add Feed Modal */}
      {showAddFeedModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-xl shadow-2xl max-w-md w-full">
            <div className="p-6 border-b border-gray-200">
              <div className="flex items-center justify-between">
                <h3 className="text-2xl font-bold text-gray-900">Add TI Feed</h3>
                <button
                  onClick={() => {
                    setShowAddFeedModal(false);
                    setFeedId('');
                    setFeedApiKey('');
                  }}
                  className="p-2 hover:bg-gray-100 rounded-lg transition-all"
                >
                  <X className="h-5 w-5" />
                </button>
              </div>
            </div>

            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Feed ID <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={feedId}
                  onChange={(e) => setFeedId(e.target.value)}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent font-mono text-sm"
                  placeholder="Enter feed ID"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  API Key <span className="text-gray-400">(optional)</span>
                </label>
                <input
                  type="text"
                  value={feedApiKey}
                  onChange={(e) => setFeedApiKey(e.target.value)}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent font-mono text-sm"
                  placeholder="Enter API key for private feeds"
                />
                <p className="text-xs text-gray-500 mt-1">
                  Only required for private feeds
                </p>
              </div>

              <div className="flex gap-4 pt-2">
                <button
                  onClick={handleAttachFeed}
                  disabled={!feedId.trim()}
                  className="px-6 py-2 bg-gradient-to-r from-purple-600 to-violet-600 text-white rounded-lg hover:from-purple-700 hover:to-violet-700 transition-all shadow-md hover:shadow-lg font-semibold disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Add Feed
                </button>
                <button
                  onClick={() => {
                    setShowAddFeedModal(false);
                    setFeedId('');
                    setFeedApiKey('');
                  }}
                  className="px-6 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-all font-semibold"
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Add User Modal */}
      {showAddUserModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-xl shadow-2xl max-w-md w-full">
            <div className="p-6 border-b border-gray-200">
              <div className="flex items-center justify-between">
                <h3 className="text-2xl font-bold text-gray-900">Add User</h3>
                <button
                  onClick={() => setShowAddUserModal(false)}
                  className="p-2 hover:bg-gray-100 rounded-lg transition-all"
                >
                  <X className="h-5 w-5" />
                </button>
              </div>
            </div>

            <form onSubmit={handleAddUser} className="p-6">
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-1">User ID</label>
                <input
                  type="text"
                  value={newUserId}
                  onChange={(e) => setNewUserId(e.target.value)}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                  placeholder="Enter user ID"
                  required
                />
              </div>

              <div className="flex gap-4">
                <button
                  type="submit"
                  className="px-6 py-2 bg-gradient-to-r from-purple-600 to-violet-600 text-white rounded-lg hover:from-purple-700 hover:to-violet-700 transition-all shadow-md hover:shadow-lg font-semibold"
                >
                  Add User
                </button>
                <button
                  type="button"
                  onClick={() => setShowAddUserModal(false)}
                  className="px-6 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-all font-semibold"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}