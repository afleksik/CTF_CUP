import { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { gatewayService } from '../services/gatewayService';
import type { VirtualService } from '../types';
import { Plus, Server, LogOut, Shield, Zap, Activity } from 'lucide-react';

export default function VirtualServicesPage() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [services, setServices] = useState<VirtualService[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!user) {
      navigate('/');
      return;
    }
    loadServices();
  }, [user, navigate]);

  const loadServices = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await gatewayService.getVirtualServices();
      setServices(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load virtual services');
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = () => {
    logout();
    navigate('/');
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-purple-50 via-violet-50 to-indigo-50 flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-purple-50 via-violet-50 to-indigo-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="bg-white rounded-xl shadow-lg p-6 mb-8 border border-purple-100">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-gradient-to-br from-purple-500 to-violet-600 rounded-xl">
                <Server className="h-8 w-8 text-white" />
              </div>
              <div>
                <h1 className="text-3xl font-bold text-gray-900">Virtual Services</h1>
                <p className="text-gray-600 mt-1">Intelligent API Gateway Management</p>
              </div>
            </div>
            <button
              onClick={handleLogout}
              className="inline-flex items-center gap-2 px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
            >
              <LogOut className="h-4 w-4" />
              Logout
            </button>
          </div>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-800">
            {error}
          </div>
        )}

        {/* Stats Cards */}
        <div className="grid md:grid-cols-3 gap-6 mb-8">
          <div className="bg-white rounded-xl shadow-md p-6 border border-purple-100">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">Total Services</p>
                <p className="text-3xl font-bold text-purple-600 mt-1">{services.length}</p>
              </div>
              <div className="p-3 bg-purple-100 rounded-lg">
                <Server className="h-6 w-6 text-purple-600" />
              </div>
            </div>
          </div>

          <div className="bg-white rounded-xl shadow-md p-6 border border-green-100">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">Active Services</p>
                <p className="text-3xl font-bold text-green-600 mt-1">
                  {services.filter(s => s.is_active).length}
                </p>
              </div>
              <div className="p-3 bg-green-100 rounded-lg">
                <Activity className="h-6 w-6 text-green-600" />
              </div>
            </div>
          </div>

          <div className="bg-white rounded-xl shadow-md p-6 border border-violet-100">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">Protected Services</p>
                <p className="text-3xl font-bold text-violet-600 mt-1">
                  {services.filter(s => s.ti_mode !== 'disabled').length}
                </p>
              </div>
              <div className="p-3 bg-violet-100 rounded-lg">
                <Shield className="h-6 w-6 text-violet-600" />
              </div>
            </div>
          </div>
        </div>

        {/* Services Grid */}
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {services.length > 0 && (
            /* Create New Service Card */
            <Link
              to="/services/new"
              className="flex flex-col items-center justify-center p-8 bg-white border-2 border-dashed border-purple-300 rounded-xl hover:border-purple-500 hover:bg-purple-50 transition-all cursor-pointer group"
            >
              <div className="p-4 bg-purple-100 rounded-full group-hover:bg-purple-200 transition-colors mb-4">
                <Plus className="h-8 w-8 text-purple-600" />
              </div>
              <p className="text-gray-900 font-semibold text-lg mb-1">Create New Service</p>
              <p className="text-gray-500 text-sm text-center">
                Set up a new virtual service with TI protection
              </p>
            </Link>
          )}

          {/* Service Cards */}
          {services.map((service) => (
            <Link
              key={service.id}
              to={`/services/${service.id}`}
              className="block p-6 bg-white rounded-xl shadow-md hover:shadow-xl transition-all border border-purple-100 hover:border-purple-300 group"
            >
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className={`p-2 rounded-lg ${
                    service.is_active
                      ? 'bg-gradient-to-br from-purple-500 to-violet-600'
                      : 'bg-gray-300'
                  }`}>
                    <Server className="h-6 w-6 text-white" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <h3 className="font-bold text-gray-900 group-hover:text-purple-600 transition-colors truncate">
                      {service.name}
                    </h3>
                    <p className="text-sm text-gray-500 font-mono truncate">/{service.slug}</p>
                  </div>
                </div>
              </div>

              <div className="space-y-3">
                {/* Status Badge */}
                <div className="flex items-center gap-2">
                  <span className={`flex-shrink-0 w-2 h-2 rounded-full ${
                    service.is_active ? 'bg-green-500' : 'bg-gray-400'
                  }`}></span>
                  <span className="text-sm text-gray-600">
                    {service.is_active ? 'Active' : 'Inactive'}
                  </span>
                </div>

                {/* TI Mode Badge */}
                <div className="flex items-center gap-2">
                  <Shield className={`h-4 w-4 ${
                    service.ti_mode === 'block' ? 'text-red-500' :
                    service.ti_mode === 'monitor' ? 'text-yellow-500' :
                    'text-gray-400'
                  }`} />
                  <span className="text-sm text-gray-600 capitalize">
                    TI: {service.ti_mode}
                  </span>
                </div>

                {/* Rate Limiting */}
                {service.rate_limit_enabled && (
                  <div className="flex items-center gap-2">
                    <Zap className="h-4 w-4 text-blue-500" />
                    <span className="text-sm text-gray-600">
                      {service.rate_limit_requests}req/{service.rate_limit_window_sec}s
                    </span>
                  </div>
                )}

                {/* Backend URL */}
                <div className="pt-2 border-t border-gray-100">
                  <p className="text-xs text-gray-500">Backend:</p>
                  <p className="text-xs text-gray-900 font-mono truncate mt-1">
                    {service.backend_url}
                  </p>
                </div>
              </div>
            </Link>
          ))}
        </div>

        {/* Empty State */}
        {services.length === 0 && !error && (
          <div className="text-center py-16 bg-white rounded-xl shadow-md border border-purple-100">
            <div className="p-4 bg-purple-100 rounded-full w-20 h-20 mx-auto mb-6 flex items-center justify-center">
              <Server className="h-10 w-10 text-purple-600" />
            </div>
            <h3 className="text-2xl font-bold text-gray-900 mb-2">No Virtual Services Yet</h3>
            <p className="text-gray-600 mb-6 max-w-md mx-auto">
              Create your first virtual service to start protecting your APIs with threat intelligence
            </p>
            <Link
              to="/services/new"
              className="inline-flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-purple-600 to-violet-600 text-white rounded-lg hover:from-purple-700 hover:to-violet-700 transition-all shadow-md hover:shadow-lg font-semibold"
            >
              <Plus className="h-5 w-5" />
              Create Your First Service
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}