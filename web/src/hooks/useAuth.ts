import { useAuthStore } from '../stores/authStore';
import { useEffect } from 'react';

export function useAuth() {
  const store = useAuthStore();

  // Auto-fetch user on mount if token exists
  useEffect(() => {
    if (store.token && !store.user) {
      store.fetchCurrentUser().catch(() => {
        // Token invalid, will be cleared by store
      });
    }
  }, [store.token]);

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