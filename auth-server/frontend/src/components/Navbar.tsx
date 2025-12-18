import { Link, useLocation } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { Shield, Home, Users, LayoutDashboard, UserCircle, LogOut } from 'lucide-react';

export function Navbar() {
  const { user, logout } = useAuth();
  const location = useLocation();

  const handleLogout = async () => {
    await logout();
  };

  const isActive = (path: string) => {
    return location.pathname === path;
  };

  const linkClasses = (path: string) =>
    `px-3 py-2 rounded-md text-sm font-medium transition-colors ${
      isActive(path)
        ? 'bg-blue-100 text-blue-700'
        : 'text-gray-700 hover:bg-gray-100'
    }`;

  return (
    <nav className="bg-white shadow-sm sticky top-0 z-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex items-center gap-8">
            <Link to="/" className="flex items-center gap-2 text-xl font-bold text-blue-600">
              <Shield className="w-6 h-6" />
              <span>HazyAuthServer</span>
            </Link>
            <div className="hidden md:flex items-center gap-2">
              <Link to="/" className={linkClasses('/')}>
                <Home className="w-4 h-4 inline mr-1" />
                Home
              </Link>
              <Link to="/users" className={linkClasses('/users')}>
                <Users className="w-4 h-4 inline mr-1" />
                Users
              </Link>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {user ? (
              <>
                <Link to="/dashboard" className={linkClasses('/dashboard')}>
                  <LayoutDashboard className="w-4 h-4 inline mr-1" />
                  Dashboard
                </Link>
                <Link to="/profile" className={linkClasses('/profile')}>
                  <UserCircle className="w-4 h-4 inline mr-1" />
                  Profile
                </Link>
                <button
                  onClick={handleLogout}
                  className="px-3 py-2 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-100 transition-colors"
                >
                  <LogOut className="w-4 h-4 inline mr-1" />
                  Logout
                </button>
              </>
            ) : (
              <>
                <Link
                  to="/login"
                  className="px-4 py-2 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-100 transition-colors"
                >
                  Sign In
                </Link>
                <Link
                  to="/register"
                  className="px-4 py-2 rounded-md text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 transition-colors"
                >
                  Sign Up
                </Link>
              </>
            )}
          </div>
        </div>
      </div>
    </nav>
  );
}
