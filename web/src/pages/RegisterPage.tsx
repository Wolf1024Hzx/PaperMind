import { useState } from 'react';
import { Link } from 'react-router-dom';
import { validateRegisterForm } from '../utils/validation';
import Button from '../components/common/Button';
import { Brain, User, Mail, Lock, CheckCircle } from 'lucide-react';
import toast from 'react-hot-toast';
import { apiPost } from '../api/client';

export default function RegisterPage() {
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [errors, setErrors] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [registerSuccess, setRegisterSuccess] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrors([]);

    const validation = validateRegisterForm(username, email, password);
    if (!validation.valid) {
      setErrors(validation.errors);
      return;
    }

    setIsLoading(true);
    try {
      await apiPost('/auth/register', { username, email, password });
      setRegisterSuccess(true);
      toast.success('注册成功');
    } catch (error: any) {
      const message = error?.data?.message || '注册失败，请稍后重试';
      setErrors([message]);
    } finally {
      setIsLoading(false);
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
          {registerSuccess ? (
            /* Success state */
            <div className="text-center">
              <CheckCircle className="w-16 h-16 text-green-500 mx-auto mb-4" />
              <h2 className="text-xl font-semibold text-gray-900 mb-2">
                注册成功！
              </h2>
              <p className="text-gray-600 mb-6">
                请使用您的账户登录
              </p>
              <Link
                to="/login"
                className="inline-flex items-center justify-center px-6 py-3 bg-primary-500 text-white rounded-lg font-medium hover:bg-primary-600 transition-colors"
              >
                前往登录
              </Link>
            </div>
          ) : (
            /* Register form */
            <>
              <h2 className="text-xl font-semibold text-gray-900 mb-6 text-center">
                创建账户
              </h2>

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
                    用户名
                  </label>
                  <div className="relative">
                    <User className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
                    <input
                      type="text"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      placeholder="3-20 个字符，字母数字下划线"
                      className="input pl-10"
                      disabled={isLoading}
                    />
                  </div>
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
                      placeholder="your@email.com"
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
                      placeholder="至少 6 个字符"
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
                  注册
                </Button>
              </form>

              {/* Login link */}
              <div className="mt-6 text-center">
                <p className="text-sm text-gray-600">
                  已有账户？{' '}
                  <Link
                    to="/login"
                    className="text-primary-500 hover:text-primary-600 font-medium"
                  >
                    立即登录
                  </Link>
                </p>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}