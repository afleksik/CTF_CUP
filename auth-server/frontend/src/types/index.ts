export interface User {
  id: string;
  username: string;
  email: string;
  bio: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface PublicUser {
  id: string;
  username: string;
  bio: string;
  created_at: string;
}

export interface LoginResponse {
  token: string;
  token_type: string;
  expires_in: number;
  user: User;
}

export interface RegisterData {
  username: string;
  email: string;
  password: string;
  bio?: string;
}

export interface LoginData {
  username: string;
  password: string;
}

export interface UpdateProfileData {
  email?: string;
  bio?: string;
}

export interface ApiError {
  error: string;
}

export interface UsersResponse {
  users: PublicUser[];
  total: number;
  limit: number;
  offset: number;
}
