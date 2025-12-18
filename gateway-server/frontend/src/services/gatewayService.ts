import type {
  VirtualService,
  VSWithFeeds,
  VirtualServiceUser,
  TIFeedInfo,
  TrafficLog,
  CreateVSData,
  UpdateVSData,
  AddUserToVSData,
  AttachTIFeedData,
  PaginatedResponse,
  TIFeed,
} from '../types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api';

class GatewayService {
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

  // Virtual Services
  async createVirtualService(data: CreateVSData): Promise<VirtualService> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/virtual-services`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
    return response.json();
  }

  async getVirtualServices(): Promise<VirtualService[]> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/virtual-services`);
    return response.json();
  }

  async getVirtualService(vsId: string): Promise<VSWithFeeds> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/virtual-services/${vsId}`);
    return response.json();
  }

  async updateVirtualService(vsId: string, data: UpdateVSData): Promise<VirtualService> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/virtual-services/${vsId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
    return response.json();
  }

  async deleteVirtualService(vsId: string): Promise<{ message: string }> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/virtual-services/${vsId}`, {
      method: 'DELETE',
    });
    return response.json();
  }

  // Virtual Service Users
  async addUserToVS(vsId: string, data: AddUserToVSData): Promise<{ message: string }> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/virtual-services/${vsId}/users`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
    return response.json();
  }

  async getVSUsers(vsId: string): Promise<VirtualServiceUser[]> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/virtual-services/${vsId}/users`);
    return response.json();
  }

  async removeUserFromVS(vsId: string, userId: string): Promise<{ message: string }> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/virtual-services/${vsId}/users/${userId}`, {
      method: 'DELETE',
    });
    return response.json();
  }

  // TI Feeds
  async getAvailableTIFeeds(): Promise<TIFeed[]> {
    // Use gateway as proxy to avoid CORS issues
    const response = await this.fetchWithAuth(`${API_BASE_URL}/ti-feeds`);
    return response.json();
  }

  async attachTIFeed(vsId: string, data: AttachTIFeedData): Promise<{ message: string }> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/virtual-services/${vsId}/ti-feeds`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
    return response.json();
  }

  async getVSFeeds(vsId: string): Promise<TIFeedInfo[]> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/virtual-services/${vsId}/ti-feeds`);
    return response.json();
  }

  async toggleTIFeed(vsId: string, feedId: string, isActive: boolean): Promise<{ message: string }> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/virtual-services/${vsId}/ti-feeds/${feedId}`, {
      method: 'PUT',
      body: JSON.stringify({ is_active: isActive }),
    });
    return response.json();
  }

  async detachTIFeed(vsId: string, feedId: string): Promise<{ message: string }> {
    const response = await this.fetchWithAuth(`${API_BASE_URL}/virtual-services/${vsId}/ti-feeds/${feedId}`, {
      method: 'DELETE',
    });
    return response.json();
  }

  // Traffic Logs
  async getTrafficLogs(
    vsId: string,
    params?: { limit?: number; offset?: number; blocked?: boolean }
  ): Promise<PaginatedResponse<TrafficLog>> {
    const queryParams = new URLSearchParams();
    if (params?.limit !== undefined) queryParams.append('limit', params.limit.toString());
    if (params?.offset !== undefined) queryParams.append('offset', params.offset.toString());
    if (params?.blocked !== undefined) queryParams.append('blocked', params.blocked.toString());

    const url = `${API_BASE_URL}/virtual-services/${vsId}/logs${queryParams.toString() ? '?' + queryParams.toString() : ''}`;
    const response = await this.fetchWithAuth(url);
    return response.json();
  }

  // User info
  async getUserInfo(userId: string): Promise<{ id: string; username: string; email: string }> {
    const authServerUrl = import.meta.env.VITE_AUTH_SERVER_URL || '/auth';
    const response = await fetch(`${authServerUrl}/users/${userId}`);
    if (!response.ok) {
      throw new Error('Failed to fetch user info');
    }
    return response.json();
  }
}

export const gatewayService = new GatewayService();