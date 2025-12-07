const API_BASE = 'http://localhost:8080/api';

interface ApiResponse<T> {
  data?: T;
  message?: string;
  error?: string;
}

interface LoginResponse {
  requires_totp: boolean;
  requires_totp_setup: boolean;
  token?: string;
  user?: User;
  temp_user_id?: number;
  totp_setup?: TOTPSetup;
}

interface TOTPSetup {
  secret: string;
  qr_code: string;
}

interface RegisterResponse {
  message: string;
  data: {
    user_id: number;
    totp_setup: TOTPSetup;
  };
}

interface TOTPSetupResponse {
  message: string;
  data: TOTPSetup;
}

export interface User {
  id: number;
  username: string;
  email: string;
  is_active: boolean;
  totp_enabled: boolean;
  created_at: string;
  updated_at: string;
  roles: Role[];
  permissions: string[];
}

export interface Role {
  id: number;
  name: string;
  description: string;
  created_at?: string;
  updated_at?: string;
  permissions?: string[];
}

export interface RoleWithPermissions extends Role {
  permissions: string[];
}

export interface APIKeyResponse {
  id: number;
  user_id: number;
  name: string;
  platform: string;
  api_key_masked: string;
  is_testnet: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateAPIKeyRequest {
  name: string;
  platform: string;
  api_key: string;
  api_secret: string;
  is_testnet: boolean;
}

export interface UpdateAPIKeyRequest {
  name?: string;
  api_key?: string;
  api_secret?: string;
  is_testnet?: boolean;
  is_active?: boolean;
}

class ApiClient {
  private token: string | null = null;

  constructor() {
    this.token = localStorage.getItem('token');
  }

  setToken(token: string | null) {
    this.token = token;
    if (token) {
      localStorage.setItem('token', token);
    } else {
      localStorage.removeItem('token');
    }
  }

  getToken(): string | null {
    return this.token;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...options.headers as Record<string, string>,
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers,
    });

    const data = await response.json();

    if (!response.ok) {
      throw new Error(data.error || 'Request failed');
    }

    return data;
  }

  // Auth endpoints
  async register(username: string, email: string, password: string): Promise<RegisterResponse> {
    return this.request('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, email, password }),
    });
  }

  async activateAccount(userId: number, code: string): Promise<ApiResponse<void>> {
    return this.request('/auth/activate', {
      method: 'POST',
      body: JSON.stringify({ user_id: userId, code }),
    });
  }

  async login(username: string, password: string): Promise<LoginResponse> {
    const response = await this.request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
    if (response.token) {
      this.setToken(response.token);
    }
    return response;
  }

  async verifyTOTP(userId: number, code: string): Promise<LoginResponse> {
    const response = await this.request<LoginResponse>('/auth/verify-totp', {
      method: 'POST',
      body: JSON.stringify({ user_id: userId, code }),
    });
    if (response.token) {
      this.setToken(response.token);
    }
    return response;
  }

  async logout(): Promise<void> {
    this.setToken(null);
  }

  async getMe(): Promise<User> {
    return this.request('/auth/me');
  }

  async changePassword(currentPassword: string, newPassword: string): Promise<ApiResponse<void>> {
    return this.request('/auth/change-password', {
      method: 'POST',
      body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
    });
  }

  // 2FA rebind endpoints
  async setupTOTPRebind(password: string): Promise<TOTPSetupResponse> {
    return this.request('/auth/totp/rebind', {
      method: 'POST',
      body: JSON.stringify({ password }),
    });
  }

  async confirmTOTPRebind(code: string): Promise<ApiResponse<void>> {
    return this.request('/auth/totp/rebind/confirm', {
      method: 'POST',
      body: JSON.stringify({ code }),
    });
  }

  async cancelTOTPRebind(): Promise<ApiResponse<void>> {
    return this.request('/auth/totp/rebind/cancel', {
      method: 'POST',
    });
  }

  // Kline endpoints
  async getSymbols(): Promise<ApiResponse<string[]>> {
    return this.request('/kline/symbols');
  }

  async getIntervals(): Promise<ApiResponse<string[]>> {
    return this.request('/kline/intervals');
  }

  // RBAC endpoints
  async listRoles(): Promise<ApiResponse<RoleWithPermissions[]>> {
    return this.request('/rbac/roles');
  }

  async getRole(id: number): Promise<ApiResponse<RoleWithPermissions>> {
    return this.request(`/rbac/roles/${id}`);
  }

  async createRole(name: string, description: string, permissions: string[]): Promise<ApiResponse<RoleWithPermissions>> {
    return this.request('/rbac/roles', {
      method: 'POST',
      body: JSON.stringify({ name, description, permissions }),
    });
  }

  async updateRole(id: number, name: string, description: string): Promise<ApiResponse<RoleWithPermissions>> {
    return this.request(`/rbac/roles/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ name, description }),
    });
  }

  async deleteRole(id: number): Promise<ApiResponse<void>> {
    return this.request(`/rbac/roles/${id}`, {
      method: 'DELETE',
    });
  }

  async setRolePermissions(id: number, permissions: string[]): Promise<ApiResponse<void>> {
    return this.request(`/rbac/roles/${id}/permissions`, {
      method: 'PUT',
      body: JSON.stringify({ permissions }),
    });
  }

  async getAllPermissions(): Promise<ApiResponse<string[]>> {
    return this.request('/rbac/permissions');
  }

  async listUsers(): Promise<ApiResponse<User[]>> {
    return this.request('/rbac/users');
  }

  async getUser(id: number): Promise<ApiResponse<User>> {
    return this.request(`/rbac/users/${id}`);
  }

  async assignRole(userId: number, roleId: number): Promise<ApiResponse<void>> {
    return this.request(`/rbac/users/${userId}/roles`, {
      method: 'POST',
      body: JSON.stringify({ role_id: roleId }),
    });
  }

  async removeRole(userId: number, roleId: number): Promise<ApiResponse<void>> {
    return this.request(`/rbac/users/${userId}/roles/${roleId}`, {
      method: 'DELETE',
    });
  }

  // API Keys endpoints
  async listAPIKeys(): Promise<ApiResponse<APIKeyResponse[]>> {
    return this.request('/api-keys/');
  }

  async getAPIKey(id: number): Promise<ApiResponse<APIKeyResponse>> {
    return this.request(`/api-keys/${id}`);
  }

  async createAPIKey(req: CreateAPIKeyRequest): Promise<ApiResponse<APIKeyResponse>> {
    return this.request('/api-keys/', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async updateAPIKey(id: number, req: UpdateAPIKeyRequest): Promise<ApiResponse<APIKeyResponse>> {
    return this.request(`/api-keys/${id}`, {
      method: 'PUT',
      body: JSON.stringify(req),
    });
  }

  async deleteAPIKey(id: number): Promise<ApiResponse<void>> {
    return this.request(`/api-keys/${id}`, {
      method: 'DELETE',
    });
  }

  async getAPIKeyPlatforms(): Promise<ApiResponse<string[]>> {
    return this.request('/api-keys/platforms');
  }
}

export const api = new ApiClient();
