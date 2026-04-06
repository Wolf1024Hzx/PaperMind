import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { validateUpdateUserForm } from '../utils/validation';
import Button from '../components/common/Button';
import Modal from '../components/common/Modal';
import Spinner from '../components/common/Spinner';
import { User, Mail, Trash2, AlertTriangle } from 'lucide-react';
import toast from 'react-hot-toast';

export default function SettingsPage() {
  const navigate = useNavigate();
  const { user, updateUser, deleteUser, isLoading } = useAuth();

  const [username, setUsername] = useState(user?.username || '');
  const [email, setEmail] = useState(user?.email || '');
  const [errors, setErrors] = useState<string[]>([]);
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const handleUpdateProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrors([]);

    const validation = validateUpdateUserForm(username, email);
    if (!validation.valid) {
      setErrors(validation.errors);
      return;
    }

    try {
      await updateUser(username, email);
      toast.success('更新成功');
    } catch (error: any) {
      const message = error?.data?.message || '更新失败';
      setErrors([message]);
    }
  };

  const handleDeleteAccount = async () => {
    try {
      await deleteUser();
      toast.success('账户已删除');
      setShowDeleteModal(false);
      navigate('/login');
    } catch (error: any) {
      toast.error(error?.data?.message || '删除失败');
    }
  };

  if (!user) {
    return (
      <div className="min-h-[calc(100vh-64px)] flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="min-h-[calc(100vh-64px)] bg-gray-50 p-4 lg:p-8">
      <div className="max-w-2xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-gray-900">账户设置</h1>
          <p className="text-gray-600 mt-1">管理您的账户信息</p>
        </div>

        {/* Profile section */}
        <div className="card p-6 mb-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">个人信息</h2>

          {/* Error messages */}
          {errors.length > 0 && (
            <div className="mb-4 bg-red-50 border border-red-200 rounded-lg p-3">
              {errors.map((error, index) => (
                <p key={index} className="text-sm text-red-600">
                  {error}
                </p>
              ))}
            </div>
          )}

          <form onSubmit={handleUpdateProfile} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                用户名
              </label>
              <div className="relative">
                <User className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
                <input
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="input pl-10"
                  disabled={isLoading}
                />
              </div>
              <p className="mt-1 text-sm text-gray-500">
                3-20 个字符，只能包含字母、数字和下划线
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                邮箱
              </label>
              <div className="relative">
                <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="input pl-10"
                  disabled={isLoading}
                />
              </div>
            </div>

            <Button
              type="submit"
              variant="primary"
              isLoading={isLoading}
            >
              保存更改
            </Button>
          </form>
        </div>

        {/* Danger zone */}
        <div className="card p-6 border-red-200">
          <h2 className="text-lg font-semibold text-red-600 mb-2">危险操作</h2>
          <p className="text-sm text-gray-600 mb-4">
            删除账户后，所有数据将被永久删除且无法恢复。
          </p>
          <Button
            variant="danger"
            leftIcon={<Trash2 size={18} />}
            onClick={() => setShowDeleteModal(true)}
          >
            删除账户
          </Button>
        </div>
      </div>

      {/* Delete confirmation modal */}
      <Modal
        isOpen={showDeleteModal}
        onClose={() => setShowDeleteModal(false)}
        title="确认删除账户"
        size="sm"
      >
        <div className="flex items-start gap-3 mb-4">
          <AlertTriangle className="w-6 h-6 text-red-500 flex-shrink-0 mt-0.5" />
          <p className="text-gray-700">
            您确定要删除账户吗？此操作不可撤销，所有论文、对话记录都将被永久删除。
          </p>
        </div>
        <div className="flex gap-3">
          <Button
            variant="secondary"
            onClick={() => setShowDeleteModal(false)}
          >
            取消
          </Button>
          <Button
            variant="danger"
            isLoading={isLoading}
            onClick={handleDeleteAccount}
          >
            确认删除
          </Button>
        </div>
      </Modal>
    </div>
  );
}