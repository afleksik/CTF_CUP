import { Shield, User } from 'lucide-react';
import type { Role } from '../types';

interface RoleBadgeProps {
  role: Role;
}

export function RoleBadge({ role }: RoleBadgeProps) {
  const isAdmin = role === 'admin';
  const displayRole = isAdmin ? 'Bar Manager' : 'Bartender';

  return (
    <span
      className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${
        isAdmin
          ? 'bg-blue-100 text-blue-800'
          : 'bg-green-100 text-green-800'
      }`}
    >
      {isAdmin ? <Shield className="h-3 w-3" /> : <User className="h-3 w-3" />}
      {displayRole}
    </span>
  );
}