import { useEffect } from 'react';
import { usePaperStore } from '../../stores/paperStore';
import { useUIStore } from '../../stores/uiStore';
import PaperCard from './PaperCard';
import PaperUploadModal from './PaperUpload';
import EmptyState from '../common/EmptyState';
import Spinner from '../common/Spinner';
import Button from '../common/Button';
import { FileX, Plus, RefreshCw } from 'lucide-react';
import toast from 'react-hot-toast';

export default function PaperList() {
  const { papers, isLoading, fetchPapers, deletePaper } = usePaperStore();
  const { uploadModalOpen, setUploadModalOpen } = useUIStore();

  useEffect(() => {
    fetchPapers().catch(() => {
      toast.error('获取论文列表失败');
    });
  }, [fetchPapers]);

  const handleDelete = async (id: string) => {
    try {
      await deletePaper(id);
      toast.success('删除成功');
    } catch (error: any) {
      toast.error(error?.data?.message || '删除失败');
      throw error;
    }
  };

  const handleRefresh = () => {
    fetchPapers().catch(() => {
      toast.error('刷新失败');
    });
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <>
      {/* Toolbar */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <h2 className="text-lg font-semibold text-gray-900">
            我的论文 ({papers.length})
          </h2>
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<RefreshCw size={16} />}
            onClick={handleRefresh}
          >
            刷新
          </Button>
        </div>
        <Button
          variant="primary"
          size="sm"
          leftIcon={<Plus size={18} />}
          onClick={() => setUploadModalOpen(true)}
        >
          上传论文
        </Button>
      </div>

      {/* Paper list */}
      {papers.length === 0 ? (
        <EmptyState
          icon={<FileX className="w-8 h-8 text-gray-400" />}
          title="暂无论文"
          description="上传论文后，系统会自动进行切片和向量化处理，支持 RAG 智能问答"
          action={
            <Button
              variant="primary"
              leftIcon={<Plus size={18} />}
              onClick={() => setUploadModalOpen(true)}
            >
              上传第一篇论文
            </Button>
          }
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {papers.map((paper) => (
            <PaperCard key={paper.id} paper={paper} onDelete={handleDelete} />
          ))}
        </div>
      )}

      {/* Upload modal */}
      <PaperUploadModal
        isOpen={uploadModalOpen}
        onClose={() => setUploadModalOpen(false)}
        onSuccess={() => {
          fetchPapers();
        }}
      />
    </>
  );
}