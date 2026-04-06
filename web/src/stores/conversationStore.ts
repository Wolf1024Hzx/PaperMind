import { create } from 'zustand';
import type { Conversation, Message, ChatResult, SectionType } from '../types';
import * as chatApi from '../api/chat';

interface ConversationState {
  conversations: Conversation[];
  activeConversationId: string | null;
  messages: Message[];
  isLoading: boolean;
  isSending: boolean;

  // Actions
  fetchConversations: () => Promise<void>;
  setActiveConversation: (id: string | null) => void;
  fetchMessages: (conversationId: string) => Promise<void>;
  sendMessage: (
    question: string,
    options?: {
      paperIds?: string[];
      sectionTypes?: SectionType[];
      yearFrom?: number;
    }
  ) => Promise<ChatResult>;
  deleteConversation: (id: string) => Promise<void>;
  clearMessages: () => void;
}

export const useConversationStore = create<ConversationState>((set, get) => ({
  conversations: [],
  activeConversationId: null,
  messages: [],
  isLoading: false,
  isSending: false,

  fetchConversations: async () => {
    set({ isLoading: true });
    try {
      const result = await chatApi.getConversations();
      set({ conversations: result.conversations || [], isLoading: false });
    } catch (error) {
      set({ isLoading: false });
      throw error;
    }
  },

  setActiveConversation: (id) => {
    set({ activeConversationId: id, messages: [] });
  },

  fetchMessages: async (conversationId) => {
    set({ isLoading: true });
    try {
      const result = await chatApi.getMessages(conversationId);
      set({ messages: result.messages || [], isLoading: false });
    } catch (error) {
      set({ isLoading: false });
      throw error;
    }
  },

  sendMessage: async (question, options) => {
    set({ isSending: true });

    // Add user message optimistically
    const userMessage: Message = {
      id: 'temp-user',
      conversationId: get().activeConversationId || '',
      role: 'user',
      content: question,
      createdAt: new Date().toISOString(),
    };
    set({ messages: [...get().messages, userMessage] });

    try {
      const result = await chatApi.askQuestion({
        question,
        conversationId: get().activeConversationId || undefined,
        ...options,
      });

      // Add assistant message
      const assistantMessage: Message = {
        id: 'temp-assistant',
        conversationId: result.conversationId,
        role: 'assistant',
        content: result.answer,
        references: result.references,
        tokenUsage: result.tokenUsage,
        createdAt: new Date().toISOString(),
      };

      // Update conversation ID if new
      const isNewConversation = !get().activeConversationId;
      if (isNewConversation) {
        // Create new conversation object
        const newConversation: Conversation = {
          id: result.conversationId,
          userId: '',
          title: question.slice(0, 50),
          mode: result.mode,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        };
        set({
          activeConversationId: result.conversationId,
          conversations: [newConversation, ...get().conversations],
        });
      }

      set({
        messages: [...get().messages.filter(m => m.id !== 'temp-user'), userMessage, assistantMessage],
        isSending: false,
      });

      return result;
    } catch (error) {
      // Remove optimistic user message on error
      set({
        messages: get().messages.filter(m => m.id !== 'temp-user'),
        isSending: false,
      });
      throw error;
    }
  },

  deleteConversation: async (id) => {
    try {
      await chatApi.deleteConversation(id);
      set({
        conversations: get().conversations.filter((c) => c.id !== id),
        activeConversationId:
          get().activeConversationId === id ? null : get().activeConversationId,
        messages: get().activeConversationId === id ? [] : get().messages,
      });
    } catch (error) {
      throw error;
    }
  },

  clearMessages: () => {
    set({ messages: [], activeConversationId: null });
  },
}));