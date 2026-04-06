// User types
export interface User {
  id: string;
  username: string;
  email: string;
  createdAt: string;
  updatedAt: string;
}

export interface UserProfile {
  id: string;
  username: string;
  email: string;
  createdAt: string;
  updatedAt: string;
}

// Auth types
export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

export interface LoginRequest {
  account: string;
  password: string;
}

export interface UpdateUserRequest {
  username: string;
  email: string;
}

export interface AuthResult {
  token: string;
  user: UserProfile;
}

// Paper types
export interface Paper {
  id: string;
  filename: string;
  fileSize: number;
  title: string;
  authors: string;
  year: number | null;
  venue: string;
  status: PaperStatus;
  chunkCount: number;
  createdAt: string;
  updatedAt: string;
}

export type PaperStatus = 'pending' | 'processing' | 'completed' | 'failed';

export interface UploadPaperInput {
  file: File;
  title?: string;
  authors?: string;
  year?: number;
  venue?: string;
}

// Conversation types
export interface Conversation {
  id: string;
  userId: string;
  title: string | null;
  mode: ConversationMode;
  createdAt: string;
  updatedAt: string;
}

export type ConversationMode = 'extract' | 'compare';

// Message types
export interface Message {
  id: string;
  conversationId: string;
  role: MessageRole;
  content: string;
  references?: Reference[];
  tokenUsage?: TokenUsage;
  createdAt: string;
}

export type MessageRole = 'user' | 'assistant';

export interface TokenUsage {
  input: number;
  output: number;
}

// Reference (RAG citation)
export interface Reference {
  chunkId?: string;
  ChunkID?: string; // Backend may return PascalCase
  paperId?: string;
  PaperID?: string;
  paperTitle?: string;
  PaperTitle?: string;
  authors?: string;
  Authors?: string;
  year?: number | null;
  Year?: number | null;
  sectionType?: string;
  SectionType?: string;
  sectionTitle?: string;
  SectionTitle?: string;
  pageNumber?: number | null;
  PageNumber?: number | null;
  content?: string;
  Content?: string;
  similarity?: number;
  Similarity?: number;
}

export type SectionType =
  | 'abstract'
  | 'introduction'
  | 'related_work'
  | 'method'
  | 'experiment'
  | 'discussion'
  | 'conclusion'
  | 'other';

// Chat types
export interface ChatRequest {
  conversationId?: string;
  question: string;
  paperIds?: string[];
  sectionTypes?: SectionType[];
  yearFrom?: number;
}

export interface ChatResult {
  conversationId: string;
  mode: ConversationMode;
  answer: string;
  references: Reference[];
  tokenUsage: TokenUsage;
}

// API Error
export interface ApiError {
  message: string;
  error?: string;
}