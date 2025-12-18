import { User, Calendar } from 'lucide-react';
import type { PublicUser } from '../types';

interface UserCardProps {
  user: PublicUser;
  onClick?: () => void;
}

export function UserCard({ user, onClick }: UserCardProps) {
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  const getInitials = (username: string) => {
    return username.charAt(0).toUpperCase();
  };

  return (
    <div
      onClick={onClick}
      className={`bg-white rounded-lg border border-gray-200 p-6 transition-all ${
        onClick ? 'hover:shadow-md hover:border-blue-300 cursor-pointer' : ''
      }`}
    >
      <div className="flex items-start gap-4">
        <div className="w-12 h-12 rounded-full bg-gradient-to-br from-blue-500 to-indigo-600 flex items-center justify-center text-white font-semibold text-lg flex-shrink-0">
          {getInitials(user.username)}
        </div>
        <div className="flex-1 min-w-0">
          <h3 className="font-semibold text-lg text-gray-900 truncate">
            {user.username}
          </h3>
          <p className="text-gray-600 text-sm mt-1 line-clamp-2">
            {user.bio || 'No bio provided'}
          </p>
          <div className="flex items-center gap-1 mt-3 text-gray-500 text-xs">
            <Calendar className="w-3 h-3" />
            <span>Joined {formatDate(user.created_at)}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
