import { useAuth } from '../contexts/AuthContext';
import { Link } from 'react-router-dom';
import { UserCircle, Users, Calendar, Mail, Edit, LogOut } from 'lucide-react';

export function DashboardPage() {
  const { user, logout } = useAuth();

  if (!user) return null;

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  const getAccountAge = (dateString: string) => {
    const created = new Date(dateString);
    const now = new Date();
    const days = Math.floor(
      (now.getTime() - created.getTime()) / (1000 * 60 * 60 * 24)
    );
    if (days === 0) return 'Today';
    if (days === 1) return '1 day';
    if (days < 30) return `${days} days`;
    if (days < 365) return `${Math.floor(days / 30)} months`;
    return `${Math.floor(days / 365)} years`;
  };

  const handleLogout = async () => {
    await logout();
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 py-12 px-4">
      <div className="max-w-4xl mx-auto">
        <div className="mb-8">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">
            Welcome back, {user.username}!
          </h1>
          <p className="text-gray-600">
            Manage your account and explore the community
          </p>
        </div>

        <div className="bg-white rounded-xl shadow-lg p-8 mb-8">
          <h2 className="text-2xl font-semibold text-gray-900 mb-6">
            Account Overview
          </h2>
          <div className="grid md:grid-cols-2 gap-6">
            <div className="flex items-start gap-3">
              <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center flex-shrink-0">
                <UserCircle className="w-5 h-5 text-blue-600" />
              </div>
              <div>
                <p className="text-sm text-gray-600">Username</p>
                <p className="text-lg font-semibold text-gray-900">{user.username}</p>
              </div>
            </div>

            <div className="flex items-start gap-3">
              <div className="w-10 h-10 bg-green-100 rounded-lg flex items-center justify-center flex-shrink-0">
                <Mail className="w-5 h-5 text-green-600" />
              </div>
              <div>
                <p className="text-sm text-gray-600">Email</p>
                <p className="text-lg font-semibold text-gray-900">{user.email}</p>
              </div>
            </div>

            <div className="flex items-start gap-3">
              <div className="w-10 h-10 bg-indigo-100 rounded-lg flex items-center justify-center flex-shrink-0">
                <Calendar className="w-5 h-5 text-indigo-600" />
              </div>
              <div>
                <p className="text-sm text-gray-600">Member Since</p>
                <p className="text-lg font-semibold text-gray-900">
                  {formatDate(user.created_at)}
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3">
              <div className="w-10 h-10 bg-amber-100 rounded-lg flex items-center justify-center flex-shrink-0">
                <Calendar className="w-5 h-5 text-amber-600" />
              </div>
              <div>
                <p className="text-sm text-gray-600">Account Age</p>
                <p className="text-lg font-semibold text-gray-900">
                  {getAccountAge(user.created_at)}
                </p>
              </div>
            </div>
          </div>

          {user.bio && (
            <div className="mt-6 pt-6 border-t border-gray-200">
              <p className="text-sm text-gray-600 mb-2">Bio</p>
              <p className="text-gray-900">{user.bio}</p>
            </div>
          )}
        </div>

        <div className="bg-white rounded-xl shadow-lg p-8">
          <h2 className="text-2xl font-semibold text-gray-900 mb-6">
            Quick Actions
          </h2>
          <div className="grid md:grid-cols-3 gap-4">
            <Link
              to="/profile"
              className="flex flex-col items-center justify-center p-6 border-2 border-gray-200 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-all group"
            >
              <Edit className="w-8 h-8 text-gray-600 group-hover:text-blue-600 mb-2" />
              <span className="font-semibold text-gray-900 group-hover:text-blue-600">
                Edit Profile
              </span>
            </Link>

            <Link
              to="/users"
              className="flex flex-col items-center justify-center p-6 border-2 border-gray-200 rounded-lg hover:border-green-500 hover:bg-green-50 transition-all group"
            >
              <Users className="w-8 h-8 text-gray-600 group-hover:text-green-600 mb-2" />
              <span className="font-semibold text-gray-900 group-hover:text-green-600">
                Browse Users
              </span>
            </Link>

            <button
              onClick={handleLogout}
              className="flex flex-col items-center justify-center p-6 border-2 border-gray-200 rounded-lg hover:border-red-500 hover:bg-red-50 transition-all group"
            >
              <LogOut className="w-8 h-8 text-gray-600 group-hover:text-red-600 mb-2" />
              <span className="font-semibold text-gray-900 group-hover:text-red-600">
                Logout
              </span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
