import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { authService } from '../services/authService';
import { UserCard } from '../components/UserCard';
import { LoadingSpinner } from '../components/LoadingSpinner';
import { ErrorMessage } from '../components/ErrorMessage';
import type { PublicUser } from '../types';
import { Users, Search, ChevronLeft, ChevronRight } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';

export function UsersPage() {
  const navigate = useNavigate();
  const { token, isLoading: authLoading } = useAuth();
  const [users, setUsers] = useState<PublicUser[]>([]);
  const [filteredUsers, setFilteredUsers] = useState<PublicUser[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [currentPage, setCurrentPage] = useState(0);
  const [totalUsers, setTotalUsers] = useState(0);
  const limit = 20;

  useEffect(() => {
    if (searchQuery.trim() === '') {
      setFilteredUsers(users);
    } else {
      const query = searchQuery.toLowerCase();
      setFilteredUsers(
        users.filter((user) =>
          user.username.toLowerCase().includes(query)
        )
      );
    }
  }, [searchQuery, users]);

  useEffect(() => {
    if (authLoading) {
      return;
    }

    if (!token) {
      setError('Please log in to view the users directory');
      setIsLoading(false);
      setUsers([]);
      setFilteredUsers([]);
      setTotalUsers(0);
      return;
    }

    let isActive = true;

    const loadUsers = async () => {
      setIsLoading(true);
      setError('');
      try {
        const data = await authService.getUsers(
          token,
          limit,
          currentPage * limit
        );
        if (!isActive) {
          return;
        }
        setUsers(data.users);
        setFilteredUsers(data.users);
        setTotalUsers(data.total);
      } catch (err) {
        if (!isActive) {
          return;
        }
        setError(
          err instanceof Error ? err.message : 'Failed to load users'
        );
      } finally {
        if (isActive) {
          setIsLoading(false);
        }
      }
    };

    loadUsers();

    return () => {
      isActive = false;
    };
  }, [authLoading, token, currentPage]);

  const handleUserClick = (userId: string) => {
    navigate(`/users/${userId}`);
  };

  const handleNextPage = () => {
    setCurrentPage((prev) => prev + 1);
  };

  const handlePreviousPage = () => {
    setCurrentPage((prev) => Math.max(0, prev - 1));
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 py-12 px-4">
      <div className="max-w-7xl mx-auto">
        <div className="mb-8">
          <div className="flex items-center gap-3 mb-2">
            <Users className="w-8 h-8 text-blue-600" />
            <h1 className="text-4xl font-bold text-gray-900">Users Directory</h1>
          </div>
          <p className="text-gray-600">
            Browse through our community members
          </p>
        </div>

        <div className="mb-8">
          <div className="relative">
            <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="text"
              placeholder="Search users by username..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-12 pr-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>
        </div>

        {error && (
          <div className="mb-8">
            <ErrorMessage message={error} />
          </div>
        )}

        {isLoading ? (
          <LoadingSpinner />
        ) : (
          <>
            {filteredUsers.length === 0 ? (
              <div className="text-center py-12">
                <Users className="w-16 h-16 text-gray-400 mx-auto mb-4" />
                <h3 className="text-xl font-semibold text-gray-900 mb-2">
                  No users found
                </h3>
                <p className="text-gray-600">
                  {searchQuery
                    ? 'Try adjusting your search query'
                    : 'No users available at the moment'}
                </p>
              </div>
            ) : (
              <>
                <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8">
                  {filteredUsers.map((user) => (
                    <UserCard
                      key={user.id}
                      user={user}
                      onClick={() => handleUserClick(user.id)}
                    />
                  ))}
                </div>

                {!searchQuery && (
                  <div className="flex justify-center items-center gap-4">
                    <button
                      onClick={handlePreviousPage}
                      disabled={currentPage === 0}
                      className="px-4 py-2 bg-white border border-gray-300 rounded-lg font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
                    >
                      <ChevronLeft className="w-4 h-4" />
                      Previous
                    </button>
                    <span className="text-gray-600">
                      Page {currentPage + 1}
                    </span>
                    <button
                      onClick={handleNextPage}
                      disabled={(currentPage + 1) * limit >= totalUsers}
                      className="px-4 py-2 bg-white border border-gray-300 rounded-lg font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
                    >
                      Next
                      <ChevronRight className="w-4 h-4" />
                    </button>
                  </div>
                )}
              </>
            )}
          </>
        )}
      </div>
    </div>
  );
}
