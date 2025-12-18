export type AssetType = 'spirits' | 'wine' | 'beer' | 'mixers' | 'garnishes';
export type Role = 'admin' | 'member';

export interface User {
  id: string;
  username: string;
  email: string;
}

export interface Realm {
  id: string;
  name: string;
  description: string;
  owner_user_id: string;
  created_at: string;
  updated_at: string;
  role?: Role;
}

export interface RealmUser {
  realm_id: string;
  user_id: string;
  role: Role;
  added_at: string;
  username?: string;
}

export interface UserSuggestion {
  id: string;
  username: string;
  email: string;
}

export interface Asset {
  id: string;
  realm_id: string;
  name: string;
  asset_type: AssetType;
  description: string;
  metadata: Record<string, any>;
  owner_user_id: string;
  created_at: string;
  updated_at: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
}

export interface CreateRealmData {
  name: string;
  description?: string;
}

export interface UpdateRealmData {
  name: string;
  description?: string;
}

export interface AddUserToRealmData {
  user_id: string;
  role: Role;
}

export interface CreateAssetData {
  name: string;
  asset_type: AssetType;
  description?: string;
  owner_user_id?: string;
}

export interface UpdateAssetData {
  name: string;
  asset_type: AssetType;
  description?: string;
  owner_user_id?: string;
}

export interface AuthContextType {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  login: (token: string, user: User) => void;
  logout: () => void;
}
