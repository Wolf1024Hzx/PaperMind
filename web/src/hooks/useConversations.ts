import { useConversationStore } from '../stores/conversationStore';
import { useEffect } from 'react';

export function useConversations() {
  const store = useConversationStore();

  // Auto-fetch conversations on mount
  useEffect(() => {
    if (!store.conversations.length && !store.isLoading) {
      store.fetchConversations().catch(() => {
        // Error handled by store
      });
    }
  }, []);

  return {
    conversations: store.conversations,
    activeConversationId: store.activeConversationId,
    messages: store.messages,
    isLoading: store.isLoading,
    isSending: store.isSending,
    fetchConversations: store.fetchConversations,
    setActiveConversation: store.setActiveConversation,
    fetchMessages: store.fetchMessages,
    sendMessage: store.sendMessage,
    deleteConversation: store.deleteConversation,
    clearMessages: store.clearMessages,
  };
}