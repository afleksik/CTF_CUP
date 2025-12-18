import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ArrowLeft, AlertCircle } from 'lucide-react';
import { gatewayService } from '../services/gatewayService';
import { TIMode } from '../types';

export default function CreateServicePage() {
  const navigate = useNavigate();
  const [error, setError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    slug: '',
    backend_url: '',
    require_auth: false,
    ti_mode: 'disabled' as TIMode,
    rate_limit_enabled: false,
    rate_limit_requests: 100,
    rate_limit_window_sec: 60,
    log_retention_minutes: 60,
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsSubmitting(true);

    try {
      const vs = await gatewayService.createVirtualService(formData);
      navigate(`/services/${vs.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create virtual service');
    } finally {
      setIsSubmitting(false);
    }
  };

  // Auto-generate slug from name
  const handleNameChange = (name: string) => {
    setFormData({
      ...formData,
      name,
      slug: name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, ''),
    });
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-purple-50 via-violet-50 to-indigo-50">
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Link
          to="/services"
          className="inline-flex items-center gap-2 text-gray-600 hover:text-gray-900 mb-6"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Services
        </Link>

        <div className="bg-white rounded-xl shadow-lg p-8 border border-purple-100">
          <div className="flex items-center gap-4 mb-6">
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
              <h1 className="text-3xl font-bold text-gray-900">Create Virtual Service</h1>
              <p className="text-gray-600 mt-1">Set up a new API gateway with TI protection</p>
            </div>
          </div>

          {error && (
            <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg flex items-start gap-3">
              <AlertCircle className="h-5 w-5 text-red-600 mt-0.5" />
              <div className="flex-1">
                <p className="text-sm text-red-800">{error}</p>
              </div>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Basic Information */}
            <div className="space-y-4">
              <h2 className="text-xl font-semibold text-gray-900">Basic Information</h2>

              <div>
                <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-1">
                  Service Name
                </label>
                <input
                  id="name"
                  type="text"
                  value={formData.name}
                  onChange={(e) => handleNameChange(e.target.value)}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                  placeholder="My API Service"
                  required
                />
              </div>

              <div>
                <label htmlFor="slug" className="block text-sm font-medium text-gray-700 mb-1">
                  Slug (URL path)
                </label>
                <input
                  id="slug"
                  type="text"
                  value={formData.slug}
                  onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                  placeholder="my-api-service"
                  pattern="^[a-z0-9]+(?:-[a-z0-9]+)*$"
                  required
                />
                <p className="mt-1 text-sm text-gray-500">
                  Lowercase letters, numbers, and hyphens only. Will be used as: /{formData.slug}/*
                </p>
              </div>

              <div>
                <label htmlFor="backend_url" className="block text-sm font-medium text-gray-700 mb-1">
                  Backend URL
                </label>
                <input
                  id="backend_url"
                  type="url"
                  value={formData.backend_url}
                  onChange={(e) => setFormData({ ...formData, backend_url: e.target.value })}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                  placeholder="https://api.example.com"
                  required
                />
                <p className="mt-1 text-sm text-gray-500">
                  The upstream service URL that this gateway will proxy to
                </p>
              </div>
            </div>

            {/* Security Settings */}
            <div className="space-y-4 pt-6 border-t border-gray-200">
              <h2 className="text-xl font-semibold text-gray-900">Security Settings</h2>

              <div>
                <label className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    checked={formData.require_auth}
                    onChange={(e) => setFormData({ ...formData, require_auth: e.target.checked })}
                    className="w-4 h-4 text-purple-600 border-gray-300 rounded focus:ring-purple-500"
                  />
                  <span className="text-sm font-medium text-gray-700">Require Authentication</span>
                </label>
                <p className="ml-7 text-sm text-gray-500">Only authenticated users can access this service</p>
              </div>

              <div>
                <label htmlFor="ti_mode" className="block text-sm font-medium text-gray-700 mb-1">
                  Threat Intelligence Mode
                </label>
                <select
                  id="ti_mode"
                  value={formData.ti_mode}
                  onChange={(e) => setFormData({ ...formData, ti_mode: e.target.value as TIMode })}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                >
                  <option value="disabled">Disabled - No TI checking</option>
                  <option value="monitor">Monitor - Log matches only</option>
                  <option value="block">Block - Block malicious requests</option>
                </select>
              </div>
            </div>

            {/* Rate Limiting */}
            <div className="space-y-4 pt-6 border-t border-gray-200">
              <h2 className="text-xl font-semibold text-gray-900">Rate Limiting</h2>

              <div>
                <label className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    checked={formData.rate_limit_enabled}
                    onChange={(e) => setFormData({ ...formData, rate_limit_enabled: e.target.checked })}
                    className="w-4 h-4 text-purple-600 border-gray-300 rounded focus:ring-purple-500"
                  />
                  <span className="text-sm font-medium text-gray-700">Enable Rate Limiting</span>
                </label>
              </div>

              {formData.rate_limit_enabled && (
                <div className="grid grid-cols-2 gap-4 ml-7">
                  <div>
                    <label htmlFor="rate_limit_requests" className="block text-sm font-medium text-gray-700 mb-1">
                      Max Requests
                    </label>
                    <input
                      id="rate_limit_requests"
                      type="number"
                      min="1"
                      value={formData.rate_limit_requests}
                      onChange={(e) => setFormData({ ...formData, rate_limit_requests: parseInt(e.target.value) })}
                      className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                    />
                  </div>
                  <div>
                    <label htmlFor="rate_limit_window_sec" className="block text-sm font-medium text-gray-700 mb-1">
                      Window (seconds)
                    </label>
                    <input
                      id="rate_limit_window_sec"
                      type="number"
                      min="1"
                      value={formData.rate_limit_window_sec}
                      onChange={(e) => setFormData({ ...formData, rate_limit_window_sec: parseInt(e.target.value) })}
                      className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                    />
                  </div>
                </div>
              )}
            </div>

            {/* Logging */}
            <div className="space-y-4 pt-6 border-t border-gray-200">
              <h2 className="text-xl font-semibold text-gray-900">Logging</h2>

              <div>
                <label htmlFor="log_retention_minutes" className="block text-sm font-medium text-gray-700 mb-1">
                  Log Retention (minutes)
                </label>
                <input
                  id="log_retention_minutes"
                  type="number"
                  min="0"
                  value={formData.log_retention_minutes}
                  onChange={(e) => setFormData({ ...formData, log_retention_minutes: parseInt(e.target.value) })}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                />
                <p className="mt-1 text-sm text-gray-500">
                  How long to keep traffic logs (0 = disabled)
                </p>
              </div>
            </div>

            {/* Submit Buttons */}
            <div className="flex gap-4 pt-6">
              <button
                type="submit"
                disabled={isSubmitting}
                className="px-6 py-3 bg-gradient-to-r from-purple-600 to-violet-600 text-white rounded-lg hover:from-purple-700 hover:to-violet-700 transition-all shadow-md hover:shadow-lg font-semibold disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isSubmitting ? 'Creating...' : 'Create Service'}
              </button>
              <Link
                to="/services"
                className="px-6 py-3 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-all font-semibold"
              >
                Cancel
              </Link>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}