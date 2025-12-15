const API_BASE = 'http://localhost:8887/api';

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
  temp_user_id?: string;
  totp_setup?: TOTPSetup;
}

interface TOTPSetup {
  secret: string;
  qr_code: string;
}

interface RegisterResponse {
  message: string;
  data: {
    user_id: string;
    totp_setup: TOTPSetup;
  };
}

interface TOTPSetupResponse {
  message: string;
  data: TOTPSetup;
}

export interface User {
  id: string;
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
  id: string;
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
  id: string;
  name: string;
  platform: string;
  api_key_masked: string;
  api_secret_masked?: string;
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

// BTCC Market types
export interface BTCCMarketInfo {
  name: string;
  money: string;
  stock: string;
  money_prec: number;
  stock_prec: number;
  min_amount: string;
  switch: boolean;
}

export interface BTCCMarketListResponse {
  error: null | { code: number; message: string };
  result: BTCCMarketInfo[];
  id: number;
}

// Switcher types
export interface SwitcherPair {
  enable: boolean;
}

export interface SwitcherResponse {
  id: string;
  pairs: Record<string, SwitcherPair>;
}

export interface UpdateSwitcherRequest {
  pairs: Record<string, SwitcherPair>;
}

// Setting types
export interface SettingResponse {
  id: string;
  base: string;
  quote: string;
  strategy: string;
  parameters: Record<string, any>;
}

export interface CreateSettingRequest {
  base: string;
  quote: string;
  strategy: string;
  parameters: Record<string, any>;
}

export interface UpdateSettingRequest {
  base?: string;
  quote?: string;
  strategy?: string;
  parameters?: Record<string, any>;
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

  async activateAccount(userId: string, code: string): Promise<ApiResponse<void>> {
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

  async verifyTOTP(userId: string, code: string): Promise<LoginResponse> {
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

  async getRole(id: string): Promise<ApiResponse<RoleWithPermissions>> {
    return this.request(`/rbac/roles/${id}`);
  }

  async createRole(name: string, description: string, permissions: string[]): Promise<ApiResponse<RoleWithPermissions>> {
    return this.request('/rbac/roles', {
      method: 'POST',
      body: JSON.stringify({ name, description, permissions }),
    });
  }

  async updateRole(id: string, name: string, description: string): Promise<ApiResponse<RoleWithPermissions>> {
    return this.request(`/rbac/roles/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ name, description }),
    });
  }

  async deleteRole(id: string): Promise<ApiResponse<void>> {
    return this.request(`/rbac/roles/${id}`, {
      method: 'DELETE',
    });
  }

  async setRolePermissions(id: string, permissions: string[]): Promise<ApiResponse<void>> {
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

  async getUser(id: string): Promise<ApiResponse<User>> {
    return this.request(`/rbac/users/${id}`);
  }

  async assignRole(userId: string, roleId: string): Promise<ApiResponse<void>> {
    return this.request(`/rbac/users/${userId}/roles`, {
      method: 'POST',
      body: JSON.stringify({ role_id: roleId }),
    });
  }

  async removeRole(userId: string, roleId: string): Promise<ApiResponse<void>> {
    return this.request(`/rbac/users/${userId}/roles/${roleId}`, {
      method: 'DELETE',
    });
  }

  // API Keys endpoints
  async listAPIKeys(): Promise<ApiResponse<APIKeyResponse[]>> {
    return this.request('/api-keys/');
  }

  async getAPIKey(id: string): Promise<ApiResponse<APIKeyResponse>> {
    return this.request(`/api-keys/${id}`);
  }

  async createAPIKey(req: CreateAPIKeyRequest): Promise<ApiResponse<APIKeyResponse>> {
    return this.request('/api-keys/', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async updateAPIKey(id: string, req: UpdateAPIKeyRequest): Promise<ApiResponse<APIKeyResponse>> {
    return this.request(`/api-keys/${id}`, {
      method: 'PUT',
      body: JSON.stringify(req),
    });
  }

  async deleteAPIKey(id: string): Promise<ApiResponse<void>> {
    return this.request(`/api-keys/${id}`, {
      method: 'DELETE',
    });
  }

  async getAPIKeyPlatforms(): Promise<ApiResponse<string[]>> {
    return this.request('/api-keys/platforms');
  }

  // Switcher endpoints
  async listSwitchers(): Promise<ApiResponse<SwitcherResponse[]>> {
    return this.request('/switchers/');
  }

  async getSwitcher(id: string): Promise<ApiResponse<SwitcherResponse>> {
    return this.request(`/switchers/${id}`);
  }

  async createSwitcher(req: UpdateSwitcherRequest): Promise<ApiResponse<SwitcherResponse>> {
    return this.request('/switchers/', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async updateSwitcher(id: string, req: UpdateSwitcherRequest): Promise<ApiResponse<SwitcherResponse>> {
    return this.request(`/switchers/${id}`, {
      method: 'PUT',
      body: JSON.stringify(req),
    });
  }

  async updateSwitcherPair(id: string, pair: string, enable: boolean): Promise<ApiResponse<SwitcherResponse>> {
    return this.request(`/switchers/${id}/pairs/${pair}`, {
      method: 'PUT',
      body: JSON.stringify({ enable }),
    });
  }

  async deleteSwitcher(id: string): Promise<ApiResponse<void>> {
    return this.request(`/switchers/${id}`, {
      method: 'DELETE',
    });
  }

  // Setting endpoints
  async listSettings(): Promise<ApiResponse<SettingResponse[]>> {
    return this.request('/settings/');
  }

  async getSetting(id: string): Promise<ApiResponse<SettingResponse>> {
    return this.request(`/settings/${id}`);
  }

  async createSetting(req: CreateSettingRequest): Promise<ApiResponse<SettingResponse>> {
    return this.request('/settings/', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async updateSetting(id: string, req: UpdateSettingRequest): Promise<ApiResponse<SettingResponse>> {
    return this.request(`/settings/${id}`, {
      method: 'PUT',
      body: JSON.stringify(req),
    });
  }

  async deleteSetting(id: string): Promise<ApiResponse<void>> {
    return this.request(`/settings/${id}`, {
      method: 'DELETE',
    });
  }

  // BTCC Proxy APIs
  async getBTCCMarkets(testnet: boolean = false): Promise<BTCCMarketListResponse> {
    const params = testnet ? '?testnet=true' : '';
    const response = await fetch(`${API_BASE}/btcc/markets${params}`, {
      headers: {
        'Authorization': `Bearer ${this.token}`,
      },
    });
    return response.json();
  }
}

export const api = new ApiClient();
