import { useEffect } from 'react';
import { Link } from 'react-router-dom';
import { useAuthStore } from '../stores/authStore';
import { usePaperStore } from '../stores/paperStore';
import { useConversationStore } from '../stores/conversationStore';
import Spinner from '../components/common/Spinner';
import {
  FileText,
  MessageSquare,
  Brain,
  ArrowRight,
  Upload,
  TrendingUp,
} from 'lucide-react';

export default function HomePage() {
  const { user } = useAuthStore();
  const { papers, fetchPapers, isLoading: papersLoading } = usePaperStore();
  const { conversations, fetchConversations, isLoading: convLoading } = useConversationStore();

  useEffect(() => {
    fetchPapers().catch(() => {});
    fetchConversations().catch(() => {});
  }, [fetchPapers, fetchConversations]);

  if (papersLoading || convLoading) {
    return (
      <div className="min-h-[calc(100vh-64px)] flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  const completedPapers = papers.filter((p) => p.status === 'completed');
  const processingPapers = papers.filter((p) => p.status === 'processing');

  return (
    <div className="min-h-[calc(100vh-64px)] bg-gray-50 p-4 lg:p-8">
      <div className="max-w-6xl mx-auto">
        {/* Welcome header */}
        <div className="mb-8">
          <div className="flex items-center gap-3 mb-2">
            <Brain className="w-8 h-8 text-primary-500" />
            <h1 className="text-2xl font-bold text-gray-900">
              欢迎，{user?.username || '用户'}
            </h1>
          </div>
          <p className="text-gray-600">
            PaperMind 是一个论文智能问答系统，帮助您从上传的论文中快速获取答案
          </p>
        </div>

        {/* Stats cards */}
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4 mb-8">
          <div className="card p-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-primary-50 flex items-center justify-center">
                <FileText className="w-5 h-5 text-primary-500" />
              </div>
              <div>
                <p className="text-sm text-gray-500">论文总数</p>
                <p className="text-xl font-semibold text-gray-900">{papers.length}</p>
              </div>
            </div>
          </div>

          <div className="card p-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-green-50 flex items-center justify-center">
                <TrendingUp className="w-5 h-5 text-green-500" />
              </div>
              <div>
                <p className="text-sm text-gray-500">已完成</p>
                <p className="text-xl font-semibold text-gray-900">{completedPapers.length}</p>
              </div>
            </div>
          </div>

          <div className="card p-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-yellow-50 flex items-center justify-center">
                <Spinner size="sm" className="text-yellow-500" />
              </div>
              <div>
                <p className="text-sm text-gray-500">处理中</p>
                <p className="text-xl font-semibold text-gray-900">{processingPapers.length}</p>
              </div>
            </div>
          </div>

          <div className="card p-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-purple-50 flex items-center justify-center">
                <MessageSquare className="w-5 h-5 text-purple-500" />
              </div>
              <div>
                <p className="text-sm text-gray-500">对话数</p>
                <p className="text-xl font-semibold text-gray-900">{conversations.length}</p>
              </div>
            </div>
          </div>
        </div>

        {/* Quick actions */}
        <div className="grid gap-4 sm:grid-cols-2 mb-8">
          <Link
            to="/papers"
            className="card p-6 hover:shadow-md transition-shadow group"
          >
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-xl bg-primary-50 flex items-center justify-center">
                <Upload className="w-6 h-6 text-primary-500" />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-900 group-hover:text-primary-500 transition-colors">
                  上传论文
                </h3>
                <p className="text-sm text-gray-500 mt-1">
                  上传 PDF、Markdown 或纯文本文件
                </p>
              </div>
              <ArrowRight className="w-5 h-5 text-gray-400 group-hover:text-primary-500 transition-colors" />
            </div>
          </Link>

          <Link
            to="/chat"
            className="card p-6 hover:shadow-md transition-shadow group"
          >
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-xl bg-purple-50 flex items-center justify-center">
                <MessageSquare className="w-6 h-6 text-purple-500" />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-900 group-hover:text-purple-500 transition-colors">
                  开始问答
                </h3>
                <p className="text-sm text-gray-500 mt-1">
                  基于 RAG 技术进行智能问答
                </p>
              </div>
              <ArrowRight className="w-5 h-5 text-gray-400 group-hover:text-purple-500 transition-colors" />
            </div>
          </Link>
        </div>

        {/* Getting started */}
        {completedPapers.length === 0 && (
          <div className="card p-6 bg-gradient-to-r from-primary-50 to-primary-100 border-primary-200">
            <h3 className="font-semibold text-gray-900 mb-2">快速开始</h3>
            <ol className="text-sm text-gray-700 space-y-2">
              <li className="flex items-start gap-2">
                <span className="bg-primary-500 text-white w-5 h-5 rounded-full flex items-center justify-center text-xs flex-shrink-0">1</span>
                上传论文（支持 PDF、Markdown、TXT 格式）
              </li>
              <li className="flex items-start gap-2">
                <span className="bg-primary-500 text-white w-5 h-5 rounded-full flex items-center justify-center text-xs flex-shrink-0">2</span>
                等待系统完成切片和向量化处理
              </li>
              <li className="flex items-start gap-2">
                <span className="bg-primary-500 text-white w-5 h-5 rounded-full flex items-center justify-center text-xs flex-shrink-0">3</span>
                在问答页面输入问题，获取基于论文内容的智能回答
              </li>
            </ol>
          </div>
        )}
      </div>
    </div>
  );
}