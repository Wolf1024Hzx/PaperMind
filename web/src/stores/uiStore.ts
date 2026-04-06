import { create } from 'zustand';

interface UIState {
  sidebarOpen: boolean;
  filterPanelOpen: boolean;
  uploadModalOpen: boolean;

  // Actions
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  toggleFilterPanel: () => void;
  setFilterPanelOpen: (open: boolean) => void;
  setUploadModalOpen: (open: boolean) => void;
}

export const useUIStore = create<UIState>((set) => ({
  sidebarOpen: true,
  filterPanelOpen: false,
  uploadModalOpen: false,

  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
  setSidebarOpen: (open) => set({ sidebarOpen: open }),
  toggleFilterPanel: () => set((state) => ({ filterPanelOpen: !state.filterPanelOpen })),
  setFilterPanelOpen: (open) => set({ filterPanelOpen: open }),
  setUploadModalOpen: (open) => set({ uploadModalOpen: open }),
}));