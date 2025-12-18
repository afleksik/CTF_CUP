// Virtual Service types
export interface VirtualService {
  id: string;
  owner_user_id: string;
  name: string;
  slug: string;
  backend_url: string;
  is_active: boolean;
  require_auth: boolean;
  ti_mode: TIMode;
  rate_limit_enabled: boolean;
  rate_limit_requests: number;
  rate_limit_window_sec: number;
  log_retention_minutes: number;
  created_at: string;
  updated_at: string;
}

export type TIMode = 'disabled' | 'monitor' | 'block';

export interface VirtualServiceUser {
  vs_id: string;
  user_id: string;
  username?: string;
  granted_by: string;
  granted_at: string;
}

export interface VirtualServiceTIFeed {
  vs_id: string;
  feed_id: string;
  is_active: boolean;
  added_at: string;
}

export interface TIFeedInfo {
  feed_id: string;
  feed_name: string;
  is_active: boolean;
  added_at: string;
}

export interface TrafficLog {
  id: string;
  vs_id: string;
  user_id: string | null;
  client_ip: string;
  method: string;
  path: string;
  request_headers: Record<string, string[]>;
  request_body: string;
  status_code: number;
  response_headers: Record<string, string[]>;
  response_body: string;
  ioc_matches: IOCMatch[];
  blocked: boolean;
  response_time_ms: number;
  timestamp: string;
}

export interface IOCMatch {
  ioc_type: string;
  ioc_value: string;
  location: string;
  feed_id: string;
  feed_name: string;
}

export interface VSWithFeeds extends VirtualService {
  ti_feeds: TIFeedInfo[];
}

// Request/Response types
export interface CreateVSData {
  name: string;
  slug: string;
  backend_url: string;
  require_auth: boolean;
  ti_mode: TIMode;
  rate_limit_enabled: boolean;
  rate_limit_requests: number;
  rate_limit_window_sec: number;
  log_retention_minutes: number;
}

export interface UpdateVSData {
  name?: string;
  backend_url?: string;
  is_active?: boolean;
  require_auth?: boolean;
  ti_mode?: TIMode;
  rate_limit_enabled?: boolean;
  rate_limit_requests?: number;
  rate_limit_window_sec?: number;
  log_retention_minutes?: number;
}

export interface AddUserToVSData {
  user_id: string;
}

export interface AttachTIFeedData {
  feed_id: string;
  api_key?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
}

// Auth types
export interface User {
  id: string;
  username: string;
  email: string;
}

export interface AuthContextType {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  logout: () => void;
}

export interface LoginData {
  username: string;
  password: string;
}

export interface RegisterData {
  username: string;
  email: string;
  password: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

// TI Feed types (from TI server)
export interface TIFeed {
  id: string;
  name: string;
  description: string;
  is_public: boolean;
  feed_type: string;
  is_active: boolean;
  ioc_count: number;
  created_at: string;
  updated_at: string;
}
