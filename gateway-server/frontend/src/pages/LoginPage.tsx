import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { LogIn } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';

const AUTH_SERVER_URL = import.meta.env.VITE_AUTH_SERVER_URL || '/auth';

export default function LoginPage() {
  const { user, isLoading } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoading && user) {
      navigate('/services');
    }
  }, [user, isLoading, navigate]);

  const handleLogin = () => {
    // Redirect to backend OAuth entrypoint
    // Must use full path including /gateway prefix to reach gateway backend
    window.location.href = '/gateway/auth/login';
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-purple-50 via-violet-50 to-indigo-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-2xl max-w-md w-full p-8 border border-purple-100">
        <div className="flex items-center justify-center mb-6">
          <div className="p-3 bg-gradient-to-br from-purple-500 to-violet-600 rounded-xl shadow-lg">
            <LogIn className="h-10 w-10 text-white" />
          </div>
        </div>
        <h1 className="text-3xl font-bold text-gray-900 mb-2 text-center">
          Gateway Manager
        </h1>
        <p className="text-gray-600 text-center mb-8">
          Intelligent API Gateway with Threat Intelligence
        </p>

        <button
          onClick={handleLogin}
          className="w-full py-3 bg-gradient-to-r from-purple-600 to-violet-600 text-white rounded-lg font-semibold hover:from-purple-700 hover:to-violet-700 transition-all shadow-md hover:shadow-lg"
        >
          Sign In with Auth Server
        </button>

        <p className="mt-6 text-center text-sm text-gray-600">
          Need an account?{' '}
          <a
            href={`${window.location.origin}${AUTH_SERVER_URL}/register`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-purple-600 hover:text-purple-700 font-medium"
          >
            Register on Auth Server
          </a>
        </p>
      </div>
    </div>
  );
}
