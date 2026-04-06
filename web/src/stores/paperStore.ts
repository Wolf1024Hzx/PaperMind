import { create } from 'zustand';
import type { Paper } from '../types';
import * as paperApi from '../api/paper';

interface PaperState {
  papers: Paper[];
  isLoading: boolean;
  uploadProgress: number;

  // Actions
  fetchPapers: () => Promise<void>;
  uploadPaper: (
    file: File,
    metadata?: {
      title?: string;
      authors?: string;
      year?: number;
      venue?: string;
    }
  ) => Promise<Paper>;
  deletePaper: (id: string) => Promise<void>;
  getPaperById: (id: string) => Paper | undefined;
}

export const usePaperStore = create<PaperState>((set, get) => ({
  papers: [],
  isLoading: false,
  uploadProgress: 0,

  fetchPapers: async () => {
    set({ isLoading: true });
    try {
      const papers = await paperApi.listPapers();
      set({ papers: papers || [], isLoading: false });
    } catch (error) {
      set({ isLoading: false });
      throw error;
    }
  },

  uploadPaper: async (file, metadata) => {
    set({ uploadProgress: 0 });
    try {
      const paper = await paperApi.uploadPaper(file, metadata);
      set({
        papers: [...get().papers, paper],
        uploadProgress: 100,
      });
      return paper;
    } catch (error) {
      set({ uploadProgress: 0 });
      throw error;
    }
  },

  deletePaper: async (id) => {
    try {
      await paperApi.deletePaper(id);
      set({
        papers: get().papers.filter((p) => p.id !== id),
      });
    } catch (error) {
      throw error;
    }
  },

  getPaperById: (id) => {
    return get().papers.find((p) => p.id === id);
  },
}));