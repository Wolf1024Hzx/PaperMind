import { useState } from 'react';
import type { Paper } from '../../types';
import { formatFileSize, formatDate } from '../../utils/format';
import {
  PAPER_STATUS_LABELS,
  PAPER_STATUS_COLORS,
} from '../../utils/constants';
import {
  FileText,
  Trash2,
  Clock,
  CheckCircle,
  AlertCircle,
  Loader2,
} from 'lucide-react';
import Modal from '../common/Modal';
import Button from '../common/Button';

interface PaperCardProps {
  paper: Paper;
  onDelete: (id: string) => Promise<void>;
}

export default function PaperCard({ paper, onDelete }: PaperCardProps) {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      await onDelete(paper.id);
      setShowDeleteModal(false);
    } catch {
      setIsDeleting(false);
    }
  };

  const StatusIcon = {
    pending: Clock,
    processing: Loader2,
    completed: CheckCircle,
    failed: AlertCircle,
  }[paper.status] || Clock;

  return (
    <>
      <div className="card p-4 hover:shadow-md transition-shadow">
        <div className="flex items-start gap-4">
          {/* File icon */}
          <div className="w-12 h-12 rounded-lg bg-primary-50 flex items-center justify-center flex-shrink-0">
            <FileText className="w-6 h-6 text-primary-500" />
          </div>

          {/* Content */}
          <div className="flex-1 min-w-0">
            {/* Title */}
            <h3 className="font-medium text-gray-900 truncate mb-1">
              {paper.title || paper.filename}
            </h3>

            {/* Meta info */}
            <div className="flex items-center gap-2 text-sm text-gray-500 mb-2">
              <span>{formatFileSize(paper.fileSize)}</span>
              {paper.year && <span>• {paper.year}</span>}
              {paper.chunkCount > 0 && (
                <span>• {paper.chunkCount} 个切片</span>
              )}
            </div>

            {/* Authors */}
            {paper.authors && (
              <p className="text-sm text-gray-600 truncate mb-2">
                {paper.authors}
              </p>
            )}

            {/* Status and date */}
            <div className="flex items-center gap-2">
              <span className={`badge ${PAPER_STATUS_COLORS[paper.status]}`}>
                {paper.status === 'processing' ? (
                  <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                ) : (
                  <StatusIcon className="w-3 h-3 mr-1" />
                )}
                {PAPER_STATUS_LABELS[paper.status]}
              </span>
              <span className="text-xs text-gray-400">
                {formatDate(paper.createdAt)}
              </span>
            </div>
          </div>

          {/* Actions */}
          <button
            onClick={() => setShowDeleteModal(true)}
            className="p-2 rounded-lg hover:bg-gray-100 transition-colors text-gray-400 hover:text-red-500"
          >
            <Trash2 size={18} />
          </button>
        </div>
      </div>

      {/* Delete confirmation modal */}
      <Modal
        isOpen={showDeleteModal}
        onClose={() => setShowDeleteModal(false)}
        title="删除论文"
        size="sm"
      >
        <p className="text-gray-700 mb-4">
          确定要删除「{paper.title || paper.filename}」吗？此操作不可撤销。
        </p>
        <div className="flex gap-3">
          <Button
            variant="secondary"
            onClick={() => setShowDeleteModal(false)}
            disabled={isDeleting}
          >
            取消
          </Button>
          <Button
            variant="danger"
            isLoading={isDeleting}
            onClick={handleDelete}
          >
            删除
          </Button>
        </div>
      </Modal>
    </>
  );
}