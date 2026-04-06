import { Link } from 'react-router-dom';
import { Home, ArrowLeft } from 'lucide-react';
import Button from '../components/common/Button';

export default function NotFoundPage() {
  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="text-center">
        {/* 404 */}
        <div className="mb-4">
          <span className="text-6xl font-bold text-primary-500">404</span>
        </div>

        {/* Message */}
        <h1 className="text-xl font-semibold text-gray-900 mb-2">
          页面不存在
        </h1>
        <p className="text-gray-600 mb-6">
          您访问的页面可能已被删除或地址错误
        </p>

        {/* Actions */}
        <div className="flex gap-3 justify-center">
          <Button
            variant="secondary"
            leftIcon={<ArrowLeft size={18} />}
            onClick={() => window.history.back()}
          >
            返回上一页
          </Button>
          <Link to="/">
            <Button variant="primary" leftIcon={<Home size={18} />}>
              返回首页
            </Button>
          </Link>
        </div>
      </div>
    </div>
  );
}