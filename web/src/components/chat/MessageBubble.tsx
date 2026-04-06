import type { Message } from '../../types';
import ReferenceCard from './ReferenceCard';
import { User, Bot, ChevronDown } from 'lucide-react';
import { useState } from 'react';
import { formatRelativeTime } from '../../utils/format';

interface MessageBubbleProps {
  message: Message;
}

export default function MessageBubble({ message }: MessageBubbleProps) {
  const [showReferences, setShowReferences] = useState(false);
  const isUser = message.role === 'user';

  return (
    <div
      className={`flex gap-3 ${
        isUser ? 'justify-end' : 'justify-start'
      } animate-fade-in`}
    >
      {/* Avatar (assistant only) */}
      {!isUser && (
        <div className="w-8 h-8 rounded-full bg-primary-500 flex items-center justify-center flex-shrink-0">
          <Bot className="w-5 h-5 text-white" />
        </div>
      )}

      {/* Message content */}
      <div
        className={`max-w-[80%] ${isUser ? 'order-first' : ''}`}
      >
        <div
          className={`rounded-xl px-4 py-3 ${
            isUser
              ? 'bg-primary-500 text-white'
              : 'bg-white border border-gray-200 shadow-sm'
          }`}
        >
          {/* Content */}
          <p className="text-sm whitespace-pre-wrap">{message.content}</p>

          {/* References (assistant only) */}
          {!isUser && message.references && message.references.length > 0 && (
            <div className="mt-3 pt-3 border-t border-gray-100">
              <button
                onClick={() => setShowReferences(!showReferences)}
                className="flex items-center gap-1 text-xs text-primary-500 hover:text-primary-600"
              >
                <ChevronDown
                  size={14}
                  className={`transition-transform ${
                    showReferences ? 'rotate-180' : ''
                  }`}
                />
                {message.references.length} 个引用来源
              </button>

              {showReferences && (
                <div className="mt-2 space-y-2">
                  {message.references.map((ref, index) => (
                    <ReferenceCard key={(ref.chunkId || ref.ChunkID) || index} reference={ref} />
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Token usage (assistant only) */}
          {!isUser && message.tokenUsage && (
            <div className="mt-2 text-xs text-gray-400">
              Token: {message.tokenUsage.input} + {message.tokenUsage.output}
            </div>
          )}
        </div>

        {/* Timestamp */}
        <p
          className={`text-xs text-gray-400 mt-1 ${
            isUser ? 'text-right' : 'text-left'
          }`}
        >
          {formatRelativeTime(message.createdAt)}
        </p>
      </div>

      {/* Avatar (user only) */}
      {isUser && (
        <div className="w-8 h-8 rounded-full bg-gray-200 flex items-center justify-center flex-shrink-0">
          <User className="w-5 h-5 text-gray-500" />
        </div>
      )}
    </div>
  );
}