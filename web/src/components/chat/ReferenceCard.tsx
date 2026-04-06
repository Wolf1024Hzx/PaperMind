import type { Reference } from '../../types';
import {
  SECTION_TYPE_LABELS,
  SECTION_TYPE_COLORS,
} from '../../utils/constants';
import { formatSimilarity, truncateText } from '../../utils/format';
import { ExternalLink, Quote } from 'lucide-react';
import { useState } from 'react';

interface ReferenceCardProps {
  reference: Reference;
}

export default function ReferenceCard({ reference }: ReferenceCardProps) {
  const [expanded, setExpanded] = useState(false);

  // Handle both camelCase and PascalCase from backend
  const content = reference.content || reference.Content || '';
  const sectionType = reference.sectionType || reference.SectionType || 'other';
  const paperTitle = reference.paperTitle || reference.PaperTitle || '未知论文';
  const authors = reference.authors || reference.Authors || '';
  const similarity = reference.similarity || reference.Similarity || 0;
  const sectionTitle = reference.sectionTitle || reference.SectionTitle;
  const pageNumber = reference.pageNumber || reference.PageNumber;
  const year = reference.year ?? reference.Year;

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-3 hover:shadow-sm transition-shadow">
      {/* Header */}
      <div className="flex items-start gap-2 mb-2">
        <Quote className="w-4 h-4 text-primary-400 flex-shrink-0 mt-1" />
        <div className="flex-1 min-w-0">
          {/* Paper title */}
          <p className="font-medium text-gray-900 truncate text-sm">
            {paperTitle}
          </p>

          {/* Authors and year */}
          <p className="text-xs text-gray-500 truncate">
            {authors}
            {year ? ` (${year})` : ''}
          </p>
        </div>

        {/* Similarity score */}
        <span className="bg-primary-50 text-primary-600 text-xs px-2 py-1 rounded-full font-medium">
          {formatSimilarity(similarity)}
        </span>
      </div>

      {/* Section info */}
      <div className="flex items-center gap-2 mb-2">
        <span
          className={`badge ${SECTION_TYPE_COLORS[sectionType] || SECTION_TYPE_COLORS.other}`}
        >
          {SECTION_TYPE_LABELS[sectionType] || SECTION_TYPE_LABELS.other}
        </span>
        {sectionTitle && (
          <span className="text-xs text-gray-500 truncate">
            {sectionTitle}
          </span>
        )}
        {pageNumber && (
          <span className="text-xs text-gray-400">P.{pageNumber}</span>
        )}
      </div>

      {/* Content preview */}
      <div
        className={`text-sm text-gray-700 bg-gray-50 rounded p-2 ${
          expanded ? '' : 'max-h-20 overflow-hidden'
        }`}
      >
        {expanded ? content : truncateText(content, 150)}
      </div>

      {/* Expand button */}
      {content.length > 150 && (
        <button
          onClick={() => setExpanded(!expanded)}
          className="mt-2 text-xs text-primary-500 hover:text-primary-600 flex items-center gap-1"
        >
          {expanded ? '收起' : '展开全文'}
          <ExternalLink size={12} />
        </button>
      )}
    </div>
  );
}