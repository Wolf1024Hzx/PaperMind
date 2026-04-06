import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { UserProfile } from '../types';
import { STORAGE_KEYS } from '../utils/constants';
import * as authApi from '../api/auth';

interface AuthState {
  user: UserProfile | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;

  // Actions
  login: (account: string, password: string) => Promise<void>;
  register: (username: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  fetchCurrentUser: () => Promise<void>;
  updateUser: (username: string, email: string) => Promise<void>;
  deleteUser: () => Promise<void>;
  setToken: (token: string) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      isAuthenticated: false,
      isLoading: false,

      login: async (account, password) => {
        set({ isLoading: true });
        try {
          const result = await authApi.login({ account, password });
          set({
            user: result.user,
            token: result.token,
            isAuthenticated: true,
            isLoading: false,
          });
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      register: async (username, email, password) => {
        set({ isLoading: true });
        try {
          const result = await authApi.register({ username, email, password });
          set({
            user: result.user,
            token: result.token,
            isAuthenticated: true,
            isLoading: false,
          });
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      logout: async () => {
        const token = get().token;
        if (token) {
          try {
            await authApi.logout();
          } catch {
            // Ignore logout API errors, still clear local state
          }
        }
        set({
          user: null,
          token: null,
          isAuthenticated: false,
        });
      },

      fetchCurrentUser: async () => {
        set({ isLoading: true });
        try {
          const user = await authApi.getCurrentUser();
          set({ user, isLoading: false });
        } catch (error) {
          set({ isLoading: false, isAuthenticated: false, user: null, token: null });
          throw error;
        }
      },

      updateUser: async (username, email) => {
        set({ isLoading: true });
        try {
          const user = await authApi.updateCurrentUser({ username, email });
          set({ user, isLoading: false });
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      deleteUser: async () => {
        set({ isLoading: true });
        try {
          await authApi.deleteCurrentUser();
          set({
            user: null,
            token: null,
            isAuthenticated: false,
            isLoading: false,
          });
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      setToken: (token) => {
        set({ token, isAuthenticated: true });
      },
    }),
    {
      name: STORAGE_KEYS.TOKEN,
      partialize: (state) => ({
        token: state.token,
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);