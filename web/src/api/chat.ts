import { apiGet, apiPost, apiDelete } from './client';
import type {
  ChatRequest,
  ChatResult,
  Conversation,
  Message,
} from '../types';

// Ask question (RAG)
export async function askQuestion(data: ChatRequest): Promise<ChatResult> {
  return apiPost<ChatResult>('/chat', data);
}

// Get user's conversations
export async function getConversations(): Promise<{ conversations: Conversation[] }> {
  return apiGet<{ conversations: Conversation[] }>('/conversations');
}

// Get conversation messages
export async function getMessages(
  conversationId: string
): Promise<{ messages: Message[] }> {
  return apiGet<{ messages: Message[] }>(`/conversations/${conversationId}/messages`);
}

// Delete conversation
export async function deleteConversation(id: string): Promise<{ message: string }> {
  return apiDelete<{ message: string }>(`/conversations/${id}`);
}