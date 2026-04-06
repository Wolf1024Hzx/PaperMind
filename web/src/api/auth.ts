import { apiGet, apiPost, apiPut, apiDelete } from './client';
import type {
  RegisterRequest,
  LoginRequest,
  UpdateUserRequest,
  AuthResult,
  UserProfile,
} from '../types';

// Register user
export async function register(data: RegisterRequest): Promise<AuthResult> {
  return apiPost<AuthResult>('/auth/register', data);
}

// Login
export async function login(data: LoginRequest): Promise<AuthResult> {
  return apiPost<AuthResult>('/auth/login', data);
}

// Logout
export async function logout(): Promise<{ message: string }> {
  return apiPost<{ message: string }>('/auth/logout');
}

// Get current user
export async function getCurrentUser(): Promise<UserProfile> {
  return apiGet<UserProfile>('/users/me');
}

// Update current user
export async function updateCurrentUser(
  data: UpdateUserRequest
): Promise<UserProfile> {
  return apiPut<UserProfile>('/users/me', data);
}

// Delete current user
export async function deleteCurrentUser(): Promise<void> {
  return apiDelete<void>('/users/me');
}