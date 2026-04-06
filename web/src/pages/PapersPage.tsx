import PaperList from '../components/paper/PaperList';

export default function PapersPage() {
  return (
    <div className="min-h-[calc(100vh-64px)] bg-gray-50 p-4 lg:p-8">
      <div className="max-w-7xl mx-auto">
        {/* Page header */}
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-gray-900">论文管理</h1>
          <p className="text-gray-600 mt-1">
            上传和管理您的论文，系统会自动进行智能切片和向量化处理
          </p>
        </div>

        {/* Paper list */}
        <PaperList />
      </div>
    </div>
  );
}