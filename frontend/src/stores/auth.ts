import { createRoot, createSignal } from 'solid-js';
import { api, type User } from '../lib/api';

interface PendingTOTP {
  userId: string;
  username: string;
}

interface PendingTOTPSetup {
  userId: string;
  username: string;
  secret: string;
  qrCode: string;
}

function createAuthStore() {
  const [user, setUser] = createSignal<User | null>(null);
  const [isLoading, setIsLoading] = createSignal(true);
  const [error, setError] = createSignal<string | null>(null);
  const [pendingTOTP, setPendingTOTP] = createSignal<PendingTOTP | null>(null);
  const [pendingTOTPSetup, setPendingTOTPSetup] = createSignal<PendingTOTPSetup | null>(null);

  const isAuthenticated = () => user() !== null;

  const hasPermission = (permission: string) => {
    const currentUser = user();
    if (!currentUser) return false;
    return currentUser.permissions.includes(permission);
  };

  const initialize = async () => {
    const token = api.getToken();
    if (!token) {
      setIsLoading(false);
      return;
    }

    try {
      const userData = await api.getMe();
      setUser(userData);
    } catch (e) {
      api.setToken(null);
    } finally {
      setIsLoading(false);
    }
  };

  const login = async (username: string, password: string): Promise<'success' | 'totp_required' | 'totp_setup_required' | 'error'> => {
    setError(null);
    setPendingTOTP(null);
    setPendingTOTPSetup(null);
    try {
      const response = await api.login(username, password);
      if (response.requires_totp_setup && response.temp_user_id && response.totp_setup) {
        setPendingTOTPSetup({
          userId: response.temp_user_id,
          username,
          secret: response.totp_setup.secret,
          qrCode: response.totp_setup.qr_code,
        });
        return 'totp_setup_required';
      }
      if (response.requires_totp && response.temp_user_id) {
        setPendingTOTP({ userId: response.temp_user_id, username });
        return 'totp_required';
      }
      if (response.user) {
        setUser(response.user);
      }
      return 'success';
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Login failed');
      return 'error';
    }
  };

  const verifyTOTP = async (code: string) => {
    setError(null);
    const pending = pendingTOTP();
    if (!pending) {
      setError('No pending 2FA verification');
      return false;
    }
    try {
      const response = await api.verifyTOTP(pending.userId, code);
      if (response.user) {
        setUser(response.user);
      }
      setPendingTOTP(null);
      return true;
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Verification failed');
      return false;
    }
  };

  const cancelTOTP = () => {
    setPendingTOTP(null);
    setError(null);
  };

  const activateTOTP = async (code: string) => {
    setError(null);
    const pending = pendingTOTPSetup();
    if (!pending) {
      setError('No pending 2FA setup');
      return false;
    }
    try {
      await api.activateAccount(pending.userId, code);
      // After activation, user needs to login again with TOTP
      setPendingTOTPSetup(null);
      setPendingTOTP({ userId: pending.userId, username: pending.username });
      return true;
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Activation failed');
      return false;
    }
  };

  const cancelTOTPSetup = () => {
    setPendingTOTPSetup(null);
    setError(null);
  };

  const logout = async () => {
    await api.logout();
    setUser(null);
  };

  return {
    user,
    isLoading,
    error,
    pendingTOTP,
    pendingTOTPSetup,
    isAuthenticated,
    hasPermission,
    initialize,
    login,
    verifyTOTP,
    cancelTOTP,
    activateTOTP,
    cancelTOTPSetup,
    logout,
    setError,
  };
}

export const authStore = createRoot(createAuthStore);

// Alias for easier import
export const useAuth = () => authStore;
