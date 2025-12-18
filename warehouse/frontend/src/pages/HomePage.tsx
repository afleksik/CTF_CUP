import { Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { Package, Shield, Users, LayoutDashboard, UserCircle } from 'lucide-react';

export function HomePage() {
  const { user } = useAuth();

  return (
    <div className="min-h-screen bg-gradient-to-br from-orange-50 to-amber-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-20">
        <div className="text-center mb-16">
          <div className="flex justify-center mb-6">
            <div className="p-6 bg-orange-600 rounded-2xl shadow-2xl">
              <Package className="h-16 w-16 text-white" />
            </div>
          </div>
          <h1 className="text-5xl font-bold text-gray-900 mb-4">
            Hazy Warehouse
          </h1>
          <p className="text-xl text-gray-600 max-w-2xl mx-auto mb-8">
            Organize and manage your bar inventory across multiple locations with bartender and manager roles
          </p>
          {user ? (
            <div className="flex gap-4 justify-center">
              <Link
                to="/dashboard"
                className="inline-flex items-center gap-2 px-8 py-3 bg-orange-600 text-white rounded-lg font-semibold hover:bg-orange-700 transition-colors shadow-lg hover:shadow-xl"
              >
                <LayoutDashboard className="w-5 h-5" />
                Go to Dashboard
              </Link>
              <Link
                to="/realms"
                className="inline-flex items-center gap-2 px-8 py-3 bg-white text-orange-600 rounded-lg font-semibold hover:bg-gray-50 transition-colors border-2 border-orange-600"
              >
                <UserCircle className="w-5 h-5" />
                My Bars
              </Link>
            </div>
          ) : (
            <div className="flex gap-4 justify-center">
              <Link
                to="/login"
                className="px-8 py-3 bg-orange-600 text-white rounded-lg font-semibold hover:bg-orange-700 transition-colors shadow-lg hover:shadow-xl"
              >
                Get Started
              </Link>
            </div>
          )}
        </div>

        <div className="grid md:grid-cols-3 gap-8 mt-16">
          <div className="bg-white p-8 rounded-xl shadow-lg">
            <div className="p-3 bg-orange-100 rounded-lg w-fit mb-4">
              <Package className="h-8 w-8 text-orange-600" />
            </div>
            <h3 className="text-xl font-semibold text-gray-900 mb-2">
              Comprehensive Inventory
            </h3>
            <p className="text-gray-600">
              Track spirits, wines, beers, mixers, garnishes, and all bar supplies in one centralized system
            </p>
          </div>

          <div className="bg-white p-8 rounded-xl shadow-lg">
            <div className="p-3 bg-orange-100 rounded-lg w-fit mb-4">
              <Shield className="h-8 w-8 text-orange-600" />
            </div>
            <h3 className="text-xl font-semibold text-gray-900 mb-2">
              Role-Based Access
            </h3>
            <p className="text-gray-600">
              Secure permission control with bar manager and bartender roles for efficient team management
            </p>
          </div>

          <div className="bg-white p-8 rounded-xl shadow-lg">
            <div className="p-3 bg-orange-100 rounded-lg w-fit mb-4">
              <Users className="h-8 w-8 text-orange-600" />
            </div>
            <h3 className="text-xl font-semibold text-gray-900 mb-2">
              Team Collaboration
            </h3>
            <p className="text-gray-600">
              Create bars and invite bartenders to manage inventory together efficiently across all locations
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}