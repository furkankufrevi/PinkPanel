import { create } from "zustand";

export interface OpenTab {
  path: string;
  name: string;
  content: string;
  originalContent: string;
  language: string;
  isLoading: boolean;
}

interface FileManagerState {
  // Tabs
  openTabs: OpenTab[];
  activeTabPath: string | null;

  // Tree
  expandedDirs: Set<string>;
  selectedPath: string | null;

  // Layout
  sidebarWidth: number;
  sidebarVisible: boolean;

  // Upload
  isUploading: boolean;
  uploadProgress: number;

  // Actions
  openFile: (path: string, name: string, content?: string) => void;
  closeTab: (path: string) => void;
  closeOtherTabs: (path: string) => void;
  closeAllTabs: () => void;
  setActiveTab: (path: string) => void;
  updateContent: (path: string, content: string) => void;
  markSaved: (path: string) => void;
  setTabLoading: (path: string, isLoading: boolean) => void;
  toggleDir: (path: string) => void;
  setSelectedPath: (path: string | null) => void;
  setSidebarWidth: (w: number) => void;
  toggleSidebar: () => void;
  setUploading: (uploading: boolean, progress?: number) => void;
}

function detectLanguage(name: string): string {
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  const map: Record<string, string> = {
    php: "php",
    js: "javascript",
    jsx: "javascript",
    ts: "javascript",
    tsx: "javascript",
    html: "html",
    htm: "html",
    css: "css",
    scss: "css",
    json: "json",
    xml: "xml",
    svg: "xml",
    md: "markdown",
    py: "python",
    sql: "sql",
    yaml: "yaml",
    yml: "yaml",
  };
  return map[ext] ?? "plain";
}

export const useFileManager = create<FileManagerState>((set, get) => ({
  openTabs: [],
  activeTabPath: null,
  expandedDirs: new Set<string>(),
  selectedPath: null,
  sidebarWidth: 260,
  sidebarVisible: true,
  isUploading: false,
  uploadProgress: 0,

  openFile: (path, name, content) => {
    const { openTabs } = get();
    const existing = openTabs.find((t) => t.path === path);
    if (existing) {
      set({ activeTabPath: path, selectedPath: path });
      return;
    }
    const tab: OpenTab = {
      path,
      name,
      content: content ?? "",
      originalContent: content ?? "",
      language: detectLanguage(name),
      isLoading: content === undefined,
    };
    set({
      openTabs: [...openTabs, tab],
      activeTabPath: path,
      selectedPath: path,
    });
  },

  closeTab: (path) => {
    const { openTabs, activeTabPath } = get();
    const idx = openTabs.findIndex((t) => t.path === path);
    const newTabs = openTabs.filter((t) => t.path !== path);
    let newActive = activeTabPath;
    if (activeTabPath === path) {
      if (newTabs.length === 0) {
        newActive = null;
      } else if (idx >= newTabs.length) {
        newActive = newTabs[newTabs.length - 1].path;
      } else {
        newActive = newTabs[idx].path;
      }
    }
    set({ openTabs: newTabs, activeTabPath: newActive });
  },

  closeOtherTabs: (path) => {
    const { openTabs } = get();
    set({
      openTabs: openTabs.filter((t) => t.path === path),
      activeTabPath: path,
    });
  },

  closeAllTabs: () => {
    set({ openTabs: [], activeTabPath: null });
  },

  setActiveTab: (path) => {
    set({ activeTabPath: path, selectedPath: path });
  },

  updateContent: (path, content) => {
    set({
      openTabs: get().openTabs.map((t) =>
        t.path === path ? { ...t, content } : t
      ),
    });
  },

  markSaved: (path) => {
    set({
      openTabs: get().openTabs.map((t) =>
        t.path === path ? { ...t, originalContent: t.content } : t
      ),
    });
  },

  setTabLoading: (path, isLoading) => {
    set({
      openTabs: get().openTabs.map((t) =>
        t.path === path ? { ...t, isLoading } : t
      ),
    });
  },

  toggleDir: (path) => {
    const dirs = new Set(get().expandedDirs);
    if (dirs.has(path)) {
      dirs.delete(path);
    } else {
      dirs.add(path);
    }
    set({ expandedDirs: dirs });
  },

  setSelectedPath: (path) => {
    set({ selectedPath: path });
  },

  setSidebarWidth: (w) => {
    set({ sidebarWidth: Math.max(180, Math.min(500, w)) });
  },

  toggleSidebar: () => {
    set({ sidebarVisible: !get().sidebarVisible });
  },

  setUploading: (uploading, progress) => {
    set({ isUploading: uploading, uploadProgress: progress ?? 0 });
  },
}));
