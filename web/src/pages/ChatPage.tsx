import { useEffect, useState, useRef } from 'react';
import { useParams } from 'react-router-dom';
import { useConversationStore } from '../stores/conversationStore';
import { usePaperStore } from '../stores/paperStore';
import ConversationSidebar from '../components/chat/ConversationSidebar';
import ChatInput from '../components/chat/ChatInput';
import MessageBubble from '../components/chat/MessageBubble';
import FilterPanel from '../components/chat/FilterPanel';
import EmptyState from '../components/common/EmptyState';
import Spinner from '../components/common/Spinner';
import { useUIStore } from '../stores/uiStore';
import type { SectionType } from '../types';
import { MessageSquare, FileText } from 'lucide-react';
import toast from 'react-hot-toast';
import { Link } from 'react-router-dom';

export default function ChatPage() {
  const { conversationId } = useParams();
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const {
    messages,
    isSending,
    isLoading,
    activeConversationId,
    setActiveConversation,
    fetchMessages,
    sendMessage,
    clearMessages,
    fetchConversations,
  } = useConversationStore();

  const { papers, fetchPapers } = usePaperStore();
  const { sidebarOpen } = useUIStore();

  // Filter state
  const [selectedPapers, setSelectedPapers] = useState<string[]>([]);
  const [selectedSectionTypes, setSelectedSectionTypes] = useState<SectionType[]>([]);
  const [yearFrom, setYearFrom] = useState<number | undefined>();

  // Load data on mount
  useEffect(() => {
    fetchConversations().catch(() => {});
    fetchPapers().catch(() => {});
  }, [fetchConversations, fetchPapers]);

  // Load conversation messages if ID provided
  useEffect(() => {
    if (conversationId) {
      setActiveConversation(conversationId);
      fetchMessages(conversationId).catch(() => {
        toast.error('加载对话失败');
      });
    } else if (!activeConversationId) {
      clearMessages();
    }
  }, [conversationId, setActiveConversation, fetchMessages, clearMessages, activeConversationId]);

  // Scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSend = async (question: string) => {
    try {
      await sendMessage(question, {
        paperIds: selectedPapers.length > 0 ? selectedPapers : undefined,
        sectionTypes: selectedSectionTypes.length > 0 ? selectedSectionTypes : undefined,
        yearFrom,
      });
    } catch (error: any) {
      toast.error(error?.data?.message || '发送失败');
    }
  };

  const handleFilterChange = (filters: {
    paperIds?: string[];
    sectionTypes?: SectionType[];
    yearFrom?: number;
  }) => {
    setSelectedPapers(filters.paperIds || []);
    setSelectedSectionTypes(filters.sectionTypes || []);
    setYearFrom(filters.yearFrom);
  };

  // Check if there are completed papers
  const completedPapers = papers.filter((p) => p.status === 'completed');

  return (
    <div className="min-h-[calc(100vh-64px)] bg-gray-50 flex">
      {/* Sidebar */}
      <ConversationSidebar />

      {/* Main chat area */}
      <main
        className={`flex-1 flex flex-col transition-all duration-300 ${
          sidebarOpen ? 'lg:ml-64' : ''
        }`}
      >
        {/* Chat header */}
        <div className="bg-white border-b border-gray-200 px-4 py-3 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <MessageSquare className="w-5 h-5 text-primary-500" />
            <h1 className="font-medium text-gray-900">智能问答</h1>
          </div>
          {completedPapers.length > 0 && (
            <span className="text-sm text-gray-500">
              {completedPapers.length} 篇论文可用
            </span>
          )}
        </div>

        {/* Messages area */}
        <div className="flex-1 overflow-y-auto p-4">
          {/* Loading */}
          {isLoading && messages.length === 0 && (
            <div className="flex justify-center py-8">
              <Spinner size="lg" />
            </div>
          )}

          {/* No papers - show upload prompt */}
          {completedPapers.length === 0 && (
            <div className="max-w-2xl mx-auto">
              <EmptyState
                icon={<FileText className="w-8 h-8 text-gray-400" />}
                title="暂无可用论文"
                description="请先上传论文，系统会自动进行切片和向量化处理。处理后即可进行智能问答。"
                action={
                  <Link
                    to="/papers"
                    className="btn-primary"
                  >
                    前往上传论文
                  </Link>
                }
              />
            </div>
          )}

          {/* No messages */}
          {completedPapers.length > 0 && messages.length === 0 && !isLoading && (
            <div className="max-w-2xl mx-auto">
              <EmptyState
                icon={<MessageSquare className="w-8 h-8 text-gray-400" />}
                title="开始对话"
                description="输入问题，系统会从上传的论文中检索相关内容并生成回答。您可以使用筛选条件限定论文范围。"
              />
            </div>
          )}

          {/* Messages */}
          {messages.length > 0 && (
            <div className="max-w-3xl mx-auto space-y-4">
              {messages.map((msg) => (
                <MessageBubble key={msg.id} message={msg} />
              ))}
              <div ref={messagesEndRef} />
            </div>
          )}
        </div>

        {/* Input area */}
        {completedPapers.length > 0 && (
          <div className="bg-white border-t border-gray-200 p-4">
            <div className="max-w-3xl mx-auto">
              {/* Filter panel */}
              <FilterPanel
                selectedPapers={selectedPapers}
                selectedSectionTypes={selectedSectionTypes}
                yearFrom={yearFrom}
                onChange={handleFilterChange}
              />

              {/* Chat input */}
              <ChatInput
                onSend={handleSend}
                isLoading={isSending}
                placeholder="输入问题，基于上传的论文进行智能问答..."
              />
            </div>
          </div>
        )}
      </main>
    </div>
  );
}