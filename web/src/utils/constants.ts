import type { SectionType } from '../types';

export type { SectionType };

// Section type labels (Chinese)
export const SECTION_TYPE_LABELS: Record<string, string> = {
  abstract: '摘要',
  introduction: '引言',
  related_work: '相关工作',
  method: '方法',
  experiment: '实验',
  discussion: '讨论',
  conclusion: '结论',
  other: '其他',
};

// Section type colors
export const SECTION_TYPE_COLORS: Record<string, string> = {
  abstract: 'bg-purple-100 text-purple-800',
  introduction: 'bg-blue-100 text-blue-800',
  related_work: 'bg-indigo-100 text-indigo-800',
  method: 'bg-green-100 text-green-800',
  experiment: 'bg-yellow-100 text-yellow-800',
  discussion: 'bg-orange-100 text-orange-800',
  conclusion: 'bg-red-100 text-red-800',
  other: 'bg-gray-100 text-gray-800',
};

// Paper status labels
export const PAPER_STATUS_LABELS: Record<string, string> = {
  pending: '待处理',
  processing: '处理中',
  completed: '已完成',
  failed: '处理失败',
};

// Paper status colors
export const PAPER_STATUS_COLORS: Record<string, string> = {
  pending: 'bg-gray-100 text-gray-800',
  processing: 'bg-yellow-100 text-yellow-800',
  completed: 'bg-green-100 text-green-800',
  failed: 'bg-red-100 text-red-800',
};

// Local storage keys
export const STORAGE_KEYS = {
  TOKEN: 'papermind_token',
  USER: 'papermind_user',
};