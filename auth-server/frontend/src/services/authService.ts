import type {
  User,
  PublicUser,
  LoginResponse,
  RegisterData,
  LoginData,
  UpdateProfileData,
  ApiError,
  UsersResponse,
} from '../types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

class AuthService {
  private async handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
      const error: ApiError = await response.json().catch(() => ({
        error: 'An unexpected error occurred',
      }));
      throw new Error(error.error);
    }
    return response.json();
  }

  async register(data: RegisterData): Promise<User> {
    const response = await fetch(`${API_BASE_URL}/auth/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });
    return this.handleResponse<User>(response);
  }

  async login(data: LoginData): Promise<LoginResponse> {
    const response = await fetch(`${API_BASE_URL}/auth/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });
    return this.handleResponse<LoginResponse>(response);
  }

  async getProfile(token: string): Promise<User> {
    const response = await fetch(`${API_BASE_URL}/auth/profile`, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
    return this.handleResponse<User>(response);
  }

  async updateProfile(token: string, data: UpdateProfileData): Promise<User> {
    const response = await fetch(`${API_BASE_URL}/auth/profile`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(data),
    });
    return this.handleResponse<User>(response);
  }

  async logout(token: string): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/auth/logout`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
    await this.handleResponse<{ message: string }>(response);
  }

  async getUsers(
    token: string,
    limit = 10,
    offset = 0
  ): Promise<UsersResponse> {
    const response = await fetch(
      `${API_BASE_URL}/users?limit=${limit}&offset=${offset}`,
      {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      }
    );
    return this.handleResponse<UsersResponse>(response);
  }

  async getUser(token: string, userId: string): Promise<PublicUser> {
    const response = await fetch(`${API_BASE_URL}/users/${userId}`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
    return this.handleResponse<PublicUser>(response);
  }

  async getPublicKey(): Promise<{ public_key: string; algorithm: string }> {
    const response = await fetch(`${API_BASE_URL}/auth/public-key`);
    return this.handleResponse<{ public_key: string; algorithm: string }>(
      response
    );
  }

  async healthCheck(): Promise<{ status: string }> {
    const response = await fetch(`${API_BASE_URL}/health`);
    return this.handleResponse<{ status: string }>(response);
  }
}

export const authService = new AuthService();
