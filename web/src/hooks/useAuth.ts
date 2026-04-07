import { useAuthStore } from '../stores/authStore';
import { useEffect, useRef } from 'react';

export function useAuth() {
  const store = useAuthStore();
  const initialized = useRef(false);

  // Auto-fetch user on mount if token exists (only once)
  useEffect(() => {
    if (!initialized.current && store.token && !store.user) {
      initialized.current = true;
      store.fetchCurrentUser().catch(() => {
        // Token invalid, will be cleared by store
      });
    }
  }, []);

  return {
    user: store.user,
    isAuthenticated: store.isAuthenticated,
    isLoading: store.isLoading,
    login: store.login,
    register: store.register,
    logout: store.logout,
    updateUser: store.updateUser,
    deleteUser: store.deleteUser,
  };
}