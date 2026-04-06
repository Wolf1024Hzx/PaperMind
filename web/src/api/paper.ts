import { apiGet, apiDelete, apiUpload } from './client';
import type { Paper } from '../types';

// Upload paper
export async function uploadPaper(
  file: File,
  metadata?: {
    title?: string;
    authors?: string;
    year?: number;
    venue?: string;
  }
): Promise<Paper> {
  const formData = new FormData();
  formData.append('file', file);

  if (metadata?.title) formData.append('title', metadata.title);
  if (metadata?.authors) formData.append('authors', metadata.authors);
  if (metadata?.year) formData.append('year', String(metadata.year));
  if (metadata?.venue) formData.append('venue', metadata.venue);

  return apiUpload<Paper>('/papers', formData);
}

// List user's papers
export async function listPapers(): Promise<Paper[]> {
  return apiGet<Paper[]>('/papers');
}

// Get paper by ID
export async function getPaper(id: string): Promise<Paper> {
  return apiGet<Paper>(`/papers/${id}`);
}

// Delete paper
export async function deletePaper(id: string): Promise<void> {
  return apiDelete<void>(`/papers/${id}`);
}