import type {
  Realm,
  RealmUser,
  Asset,
  PaginatedResponse,
  CreateRealmData,
  UpdateRealmData,
  AddUserToRealmData,
  CreateAssetData,
  UpdateAssetData,
  UserSuggestion,
} from '../types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api';

class AssetService {
  private async fetchWithAuth(url: string, options: RequestInit = {}) {
    const token = localStorage.getItem('auth_token');

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string> || {}),
    };

    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(url, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));

      // If token is invalid or expired, logout and redirect to login
      if (response.status === 401 && (error.error?.includes('token') || error.error?.includes('Unauthorized'))) {
        localStorage.removeItem('auth_token');
        localStorage.removeItem('auth_user');
        window.location.href = '/';
      }

      throw new Error(error.error || `HTTP ${response.status}`);
    }

    return response;
  }

  // Health check
  async healthCheck(): Promise<{ status: string }> {
    const response = await fetch(`${API_BASE_URL}/health`);
    return response.json();
  }

  // User info from auth-server
  async getUserInfo(userId: string): Promise<{ id: string; username: string; email: string }> {
    const authServerUrl = import.meta.env.VITE_AUTH_SERVER_URL || '/auth';
    const response = await fetch(`${authServerUrl}/users/${userId}`);
    if (!response.ok) {
      throw new Error('Failed to fetch user info');
    }
    return response.json();
  }

  async searchUsers(query: string, limit: number = 10): Promise<UserSuggestion[]> {
    const params = new URLSearchParams({
      query,
      limit: limit.toString(),
    });
    const response = await this.fetchWithAuth(`${API_BASE_URL}/users/search?${params.toString()}`);
    const data = await response.json();
    return data.users || [];
  }

  // Realms
  async createRealm(data: CreateRealmData): Promise<Realm> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/realms`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
    return response.json();
  }

  async getRealms(): Promise<Realm[]> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/realms`);
    return response.json();
  }

  async getRealm(realmId: string): Promise<Realm> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/realms/${realmId}`);
    return response.json();
  }

  async updateRealm(realmId: string, data: UpdateRealmData): Promise<Realm> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/realms/${realmId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
    return response.json();
  }

  async deleteRealm(realmId: string): Promise<{ message: string }> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/realms/${realmId}`, {
      method: 'DELETE',
    });
    return response.json();
  }

  // Realm Users
  async addUserToRealm(realmId: string, data: AddUserToRealmData): Promise<{ message: string }> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/realms/${realmId}/users`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
    return response.json();
  }

  async getRealmUsers(realmId: string, limit: number = 20, offset: number = 0): Promise<PaginatedResponse<RealmUser>> {
    const params = new URLSearchParams({
      limit: limit.toString(),
      offset: offset.toString(),
    });
    const response = await this.fetchWithAuth(`${API_BASE_URL}/realms/${realmId}/users?${params}`);
    return response.json();
  }

  async removeUserFromRealm(realmId: string, userId: string): Promise<{ message: string }> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/realms/${realmId}/users/${userId}`, {
      method: 'DELETE',
    });
    return response.json();
  }

  // Assets
  async createAsset(realmId: string, data: CreateAssetData): Promise<Asset> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/realms/${realmId}/assets`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
    return response.json();
  }

  async getAssets(
    realmId: string,
    params?: { type?: string; search?: string; limit?: number; offset?: number }
  ): Promise<PaginatedResponse<Asset>> {
    const queryParams = new URLSearchParams();
    if (params?.type) queryParams.append('type', params.type);
    if (params?.search) queryParams.append('search', params.search);
    if (params?.limit !== undefined) queryParams.append('limit', params.limit.toString());
    if (params?.offset !== undefined) queryParams.append('offset', params.offset.toString());

    const url = `${API_BASE_URL}/realms/${realmId}/assets${queryParams.toString() ? '?' + queryParams.toString() : ''}`;
    const response = await this.fetchWithAuth(url);
    return response.json();
  }

  async getAsset(assetId: string): Promise<Asset> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/assets/${assetId}`);
    return response.json();
  }

  async updateAsset(assetId: string, data: UpdateAssetData): Promise<Asset> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/assets/${assetId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
    return response.json();
  }

  async deleteAsset(assetId: string): Promise<{ message: string }> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/assets/${assetId}`, {
      method: 'DELETE',
    });
    return response.json();
  }

  // User Asset Management
  async getUserAssets(realmId: string, userId: string): Promise<Asset[]> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/realms/${realmId}/users/${userId}/assets`);
    return response.json();
  }

  async reassignUserAssets(realmId: string, userId: string, newOwnerId: string): Promise<{ message: string }> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/realms/${realmId}/users/${userId}/assets`, {
      method: 'POST',
      body: JSON.stringify({ new_owner_id: newOwnerId }),
    });
    return response.json();
  }

  async deleteUserAssets(realmId: string, userId: string): Promise<{ message: string }> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/realms/${realmId}/users/${userId}/assets`, {
      method: 'DELETE',
    });
    return response.json();
  }
}

export const assetService = new AssetService();
