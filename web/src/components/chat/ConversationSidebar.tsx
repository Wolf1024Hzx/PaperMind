import { useConversationStore } from '../../stores/conversationStore';
import { useUIStore } from '../../stores/uiStore';
import EmptyState from '../common/EmptyState';
import Spinner from '../common/Spinner';
import Button from '../common/Button';
import Modal from '../common/Modal';
import {
  MessageSquare,
  Plus,
  Trash2,
  MessageSquarePlus,
} from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';
import { useNavigate, useParams } from 'react-router-dom';

export default function ConversationSidebar() {
  const navigate = useNavigate();
  const { conversationId: activeIdFromParams } = useParams();

  const {
    conversations,
    activeConversationId,
    isLoading,
    setActiveConversation,
    fetchMessages,
    deleteConversation,
    clearMessages,
  } = useConversationStore();

  const { sidebarOpen } = useUIStore();
  const [showDeleteModal, setShowDeleteModal] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  // Use param if available, otherwise use store state
  const activeId = activeIdFromParams || activeConversationId;

  const handleSelectConversation = async (id: string) => {
    setActiveConversation(id);
    navigate(`/chat/${id}`);
    try {
      await fetchMessages(id);
    } catch {
      toast.error('加载对话失败');
    }
  };

  const handleNewConversation = () => {
    clearMessages();
    setActiveConversation(null);
    navigate('/chat');
  };

  const handleDeleteConversation = async () => {
    if (!showDeleteModal) return;

    setIsDeleting(true);
    try {
      await deleteConversation(showDeleteModal);
      toast.success('删除成功');
      setShowDeleteModal(null);

      // If deleted active conversation, clear messages
      if (showDeleteModal === activeId) {
        clearMessages();
        navigate('/chat');
      }
    } catch (error: any) {
      toast.error(error?.data?.message || '删除失败');
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <aside
      className={`fixed top-16 left-0 bottom-0 w-64 bg-white border-r border-gray-200 overflow-y-auto transition-transform duration-300 ${
        sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'
      }`}
    >
      {/* Header */}
      <div className="p-4 border-b border-gray-200">
        <Button
          variant="primary"
          size="sm"
          leftIcon={<Plus size={16} />}
          onClick={handleNewConversation}
          className="w-full"
        >
          新对话
        </Button>
      </div>

      {/* Conversations list */}
      <div className="p-3">
        {isLoading ? (
          <div className="flex justify-center py-4">
            <Spinner size="md" />
          </div>
        ) : conversations.length === 0 ? (
          <EmptyState
            icon={<MessageSquarePlus className="w-8 h-8 text-gray-400" />}
            title="暂无对话"
            description="点击上方按钮开始新对话"
          />
        ) : (
          <div className="space-y-1">
            {conversations.map((conv) => (
              <div
                key={conv.id}
                className={`group flex items-center gap-2 p-2 rounded-lg cursor-pointer transition-colors ${
                  activeId === conv.id
                    ? 'bg-primary-50 text-primary-600'
                    : 'hover:bg-gray-50 text-gray-700'
                }`}
              >
                <button
                  onClick={() => handleSelectConversation(conv.id)}
                  className="flex items-center gap-2 flex-1 min-w-0"
                >
                  <MessageSquare size={16} />
                  <span className="truncate text-sm font-medium">
                    {conv.title || '新对话'}
                  </span>
                </button>

                <button
                  onClick={() => setShowDeleteModal(conv.id)}
                  className="p-1 rounded hover:bg-gray-200 text-gray-400 hover:text-red-500 opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Delete confirmation modal */}
      <Modal
        isOpen={showDeleteModal !== null}
        onClose={() => setShowDeleteModal(null)}
        title="删除对话"
        size="sm"
      >
        <p className="text-gray-700 mb-4">
          确定要删除此对话吗？对话中的所有消息将被删除。
        </p>
        <div className="flex gap-3">
          <Button
            variant="secondary"
            onClick={() => setShowDeleteModal(null)}
            disabled={isDeleting}
          >
            取消
          </Button>
          <Button
            variant="danger"
            isLoading={isDeleting}
            onClick={handleDeleteConversation}
          >
            删除
          </Button>
        </div>
      </Modal>
    </aside>
  );
}