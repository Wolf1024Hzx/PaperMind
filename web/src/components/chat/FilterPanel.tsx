import { useState } from 'react';
import { usePaperStore } from '../../stores/paperStore';
import { useUIStore } from '../../stores/uiStore';
import type { SectionType } from '../../types';
import { SECTION_TYPE_LABELS } from '../../utils/constants';
import Button from '../common/Button';
import {
  Filter,
  ChevronDown,
  ChevronUp,
  FileText,
  Calendar,
  X,
} from 'lucide-react';

interface FilterPanelProps {
  selectedPapers: string[];
  selectedSectionTypes: SectionType[];
  yearFrom: number | undefined;
  onChange: (filters: {
    paperIds?: string[];
    sectionTypes?: SectionType[];
    yearFrom?: number;
  }) => void;
}

export default function FilterPanel({
  selectedPapers,
  selectedSectionTypes,
  yearFrom,
  onChange,
}: FilterPanelProps) {
  const { papers } = usePaperStore();
  const { filterPanelOpen, toggleFilterPanel } = useUIStore();
  const [yearInput, setYearInput] = useState(yearFrom?.toString() || '');

  // Only show completed papers in filter
  const completedPapers = papers.filter((p) => p.status === 'completed');

  const handlePaperToggle = (paperId: string) => {
    const newSelected = selectedPapers.includes(paperId)
      ? selectedPapers.filter((id) => id !== paperId)
      : [...selectedPapers, paperId];

    onChange({
      paperIds: newSelected.length > 0 ? newSelected : undefined,
    });
  };

  const handleSectionTypeToggle = (type: SectionType) => {
    const newSelected = selectedSectionTypes.includes(type)
      ? selectedSectionTypes.filter((t) => t !== type)
      : [...selectedSectionTypes, type];

    onChange({
      sectionTypes: newSelected.length > 0 ? newSelected : undefined,
    });
  };

  const handleYearChange = () => {
    const year = parseInt(yearInput);
    if (year > 1900 && year <= 2100) {
      onChange({ yearFrom: year });
    } else if (yearInput === '') {
      onChange({ yearFrom: undefined });
    }
  };

  const clearFilters = () => {
    onChange({
      paperIds: undefined,
      sectionTypes: undefined,
      yearFrom: undefined,
    });
    setYearInput('');
  };

  const hasFilters =
    selectedPapers.length > 0 ||
    selectedSectionTypes.length > 0 ||
    yearFrom !== undefined;

  if (completedPapers.length === 0) {
    return null;
  }

  return (
    <div className="mb-4">
      {/* Toggle button */}
      <Button
        variant="outline"
        size="sm"
        leftIcon={<Filter size={16} />}
        rightIcon={filterPanelOpen ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
        onClick={toggleFilterPanel}
      >
        筛选条件
        {hasFilters && (
          <span className="ml-1 bg-primary-100 text-primary-600 text-xs px-1.5 py-0.5 rounded-full">
            {selectedPapers.length + selectedSectionTypes.length + (yearFrom ? 1 : 0)}
          </span>
        )}
      </Button>

      {/* Filter panel */}
      {filterPanelOpen && (
        <div className="mt-3 bg-white border border-gray-200 rounded-xl p-4 animate-slide-up">
          {/* Clear filters */}
          {hasFilters && (
            <div className="flex justify-end mb-3">
              <Button
                variant="ghost"
                size="sm"
                leftIcon={<X size={14} />}
                onClick={clearFilters}
              >
                清除筛选
              </Button>
            </div>
          )}

          {/* Paper selection */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              <FileText size={16} className="inline mr-1" />
              选择论文
            </label>
            <div className="flex flex-wrap gap-2">
              {completedPapers.map((paper) => (
                <button
                  key={paper.id}
                  onClick={() => handlePaperToggle(paper.id)}
                  className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                    selectedPapers.includes(paper.id)
                      ? 'bg-primary-100 text-primary-600 border border-primary-300'
                      : 'bg-gray-50 text-gray-600 border border-gray-200 hover:bg-gray-100'
                  }`}
                >
                  {paper.title || paper.filename}
                </button>
              ))}
            </div>
          </div>

          {/* Section type selection */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              章节类型
            </label>
            <div className="flex flex-wrap gap-2">
              {(Object.keys(SECTION_TYPE_LABELS) as SectionType[]).map((type) => (
                <button
                  key={type}
                  onClick={() => handleSectionTypeToggle(type)}
                  className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                    selectedSectionTypes.includes(type)
                      ? 'bg-primary-100 text-primary-600 border border-primary-300'
                      : 'bg-gray-50 text-gray-600 border border-gray-200 hover:bg-gray-100'
                  }`}
                >
                  {SECTION_TYPE_LABELS[type]}
                </button>
              ))}
            </div>
          </div>

          {/* Year filter */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              <Calendar size={16} className="inline mr-1" />
              发表年份（从）
            </label>
            <input
              type="number"
              value={yearInput}
              onChange={(e) => setYearInput(e.target.value)}
              onBlur={handleYearChange}
              onKeyDown={(e) => e.key === 'Enter' && handleYearChange()}
              placeholder="例如: 2020"
              className="input w-32"
              min="1900"
              max="2100"
            />
          </div>
        </div>
      )}
    </div>
  );
}