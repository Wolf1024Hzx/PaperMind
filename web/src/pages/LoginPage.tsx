import { useState } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { validateLoginForm } from '../utils/validation';
import Button from '../components/common/Button';
import { Brain, User, Lock } from 'lucide-react';
import toast from 'react-hot-toast';

export default function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { login, isLoading } = useAuth();

  const [account, setAccount] = useState('');
  const [password, setPassword] = useState('');
  const [errors, setErrors] = useState<string[]>([]);

  const from = location.state?.from?.pathname || '/';

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrors([]);

    const validation = validateLoginForm(account, password);
    if (!validation.valid) {
      setErrors(validation.errors);
      return;
    }

    try {
      await login(account, password);
      toast.success('登录成功');
      navigate(from, { replace: true });
    } catch (error: any) {
      const message = error?.data?.message || '登录失败，请检查用户名/邮箱和密码';
      setErrors([message]);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-primary-50 to-primary-100 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Logo */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-2 mb-4">
            <Brain className="w-10 h-10 text-primary-500" />
            <span className="text-2xl font-bold text-gray-900">PaperMind</span>
          </div>
          <p className="text-gray-600">论文智能问答系统</p>
        </div>

        {/* Form card */}
        <div className="bg-white rounded-xl shadow-lg p-8">
          <h2 className="text-xl font-semibold text-gray-900 mb-6 text-center">
            登录账户
          </h2>

          {/* Test account hint */}
          <div className="mb-4 bg-blue-50 border border-blue-200 rounded-lg p-3">
            <p className="text-sm text-blue-600 text-center">
              测试账号：<span className="font-medium">test</span> / 密码：<span className="font-medium">123456</span>
              <button
                type="button"
                onClick={() => { setAccount('test'); setPassword('123456'); }}
                className="ml-2 text-blue-500 hover:text-blue-700 underline"
              >
                一键填充
              </button>
            </p>
          </div>

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

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                用户名 / 邮箱
              </label>
              <div className="relative">
                <User className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
                <input
                  type="text"
                  value={account}
                  onChange={(e) => setAccount(e.target.value)}
                  placeholder="输入用户名或邮箱"
                  className="input pl-10"
                  disabled={isLoading}
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                密码
              </label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="输入密码"
                  className="input pl-10"
                  disabled={isLoading}
                />
              </div>
            </div>

            <Button
              type="submit"
              variant="primary"
              className="w-full"
              isLoading={isLoading}
            >
              登录
            </Button>
          </form>

          {/* Register link */}
          <div className="mt-6 text-center">
            <p className="text-sm text-gray-600">
              还没有账户？{' '}
              <Link
                to="/register"
                className="text-primary-500 hover:text-primary-600 font-medium"
              >
                立即注册
              </Link>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}