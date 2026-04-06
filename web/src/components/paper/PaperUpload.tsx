import { useState } from 'react';
import { usePaperStore } from '../../stores/paperStore';
import Modal from '../common/Modal';
import FileUpload from '../common/FileUpload';
import Button from '../common/Button';
import Input from '../common/Input';
import { Upload } from 'lucide-react';
import toast from 'react-hot-toast';
import type { Paper } from '../../types';

interface PaperUploadModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: (paper: Paper) => void;
}

export default function PaperUploadModal({
  isOpen,
  onClose,
  onSuccess,
}: PaperUploadModalProps) {
  const { uploadPaper } = usePaperStore();

  const [file, setFile] = useState<File | null>(null);
  const [title, setTitle] = useState('');
  const [authors, setAuthors] = useState('');
  const [year, setYear] = useState('');
  const [venue, setVenue] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleFileSelect = (selectedFile: File) => {
    setFile(selectedFile);
    // Auto-fill title from filename
    if (!title) {
      setTitle(selectedFile.name.replace(/\.[^/.]+$/, ''));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!file) {
      toast.error('请先选择文件');
      return;
    }

    setIsSubmitting(true);

    try {
      const paper = await uploadPaper(file, {
        title: title || undefined,
        authors: authors || undefined,
        year: year ? parseInt(year) : undefined,
        venue: venue || undefined,
      });

      toast.success('论文上传成功');

      // Reset form
      setFile(null);
      setTitle('');
      setAuthors('');
      setYear('');
      setVenue('');
      setIsSubmitting(false);

      onSuccess?.(paper);
      onClose();
    } catch (error: any) {
      setIsSubmitting(false);
      toast.error(error?.data?.message || '上传失败');
    }
  };

  const handleClose = () => {
    if (!isSubmitting) {
      setFile(null);
      setTitle('');
      setAuthors('');
      setYear('');
      setVenue('');
      onClose();
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="上传论文" size="lg">
      <form onSubmit={handleSubmit} className="space-y-4">
        {/* File upload */}
        <FileUpload
          onFileSelect={handleFileSelect}
          disabled={isSubmitting}
        />

        {/* Metadata fields */}
        {file && (
          <>
            <div className="border-t border-gray-200 pt-4 mt-4">
              <p className="text-sm font-medium text-gray-700 mb-3">
                论文元数据（可选）
              </p>
            </div>

            <Input
              label="标题"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="论文标题"
              disabled={isSubmitting}
              helperText="默认使用文件名"
            />

            <Input
              label="作者"
              value={authors}
              onChange={(e) => setAuthors(e.target.value)}
              placeholder="作者姓名，多个作者用逗号分隔"
              disabled={isSubmitting}
            />

            <Input
              label="年份"
              type="number"
              value={year}
              onChange={(e) => setYear(e.target.value)}
              placeholder="发表年份"
              disabled={isSubmitting}
              min="1900"
              max="2100"
            />

            <Input
              label="发表场所"
              value={venue}
              onChange={(e) => setVenue(e.target.value)}
              placeholder="期刊/会议名称"
              disabled={isSubmitting}
            />
          </>
        )}

        {/* Submit button */}
        <div className="flex justify-end gap-3 pt-4">
          <Button
            type="button"
            variant="secondary"
            onClick={handleClose}
            disabled={isSubmitting}
          >
            取消
          </Button>
          <Button
            type="submit"
            variant="primary"
            leftIcon={isSubmitting ? undefined : <Upload size={18} />}
            isLoading={isSubmitting}
            disabled={!file}
          >
            上传
          </Button>
        </div>
      </form>
    </Modal>
  );
}