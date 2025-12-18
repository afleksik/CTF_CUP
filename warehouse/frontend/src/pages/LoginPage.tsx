import { LogIn } from 'lucide-react';

const AUTH_SERVER_URL = import.meta.env.VITE_AUTH_SERVER_URL || '/auth';

export function LoginPage() {
  const handleLogin = () => {
    // Redirect to backend OAuth entrypoint
    // Must use full path including /warehouse prefix to reach warehouse backend
    window.location.href = '/warehouse/auth/login';
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-orange-50 to-amber-50 flex items-center justify-center px-4">
      <div className="max-w-md w-full">
        <div className="bg-white rounded-xl shadow-xl p-8">
          <div className="text-center mb-8">
            <div className="inline-flex p-3 bg-orange-600 rounded-lg mb-4">
              <LogIn className="h-8 w-8 text-white" />
            </div>
            <h2 className="text-3xl font-bold text-gray-900">Sign In</h2>
            <p className="text-gray-600 mt-2">
              Sign in to access Hazy Warehouse
            </p>
          </div>

          <button
            onClick={handleLogin}
            className="w-full py-3 bg-orange-600 text-white rounded-lg font-semibold hover:bg-orange-700 transition-colors shadow-lg hover:shadow-xl"
          >
            Sign In with OAuth
          </button>

          <p className="mt-6 text-center text-sm text-gray-600">
            Need an account?{' '}
            <a
              href={`${window.location.origin}${AUTH_SERVER_URL}/register`}
              target="_blank"
              rel="noopener noreferrer"
              className="text-orange-600 hover:text-orange-700 font-medium"
            >
              Register on Auth Server
            </a>
          </p>
        </div>
      </div>
    </div>
  );
}
