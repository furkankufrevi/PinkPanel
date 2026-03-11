import { useState, useEffect, useCallback, useRef } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { toast } from "sonner";
import {
  listFiles,
  saveFile,
  deleteFile,
  renameFile,
  createDirectory,
  uploadFiles,
  downloadFile,
  compressFiles,
  extractArchive,
} from "@/api/files";
import { useFileManager } from "@/stores/file-manager";
import {
  FileTree,
  EditorTabs,
  CodeEditor,
  FileToolbar,
  UploadZone,
  ContextMenu,
  QuickOpen,
  CompressDialog,
  WelcomePanel,
  ImagePreview,
} from "./file-manager";
import type { FileEntry } from "@/types/files";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import { Loader2 } from "lucide-react";

export function DomainFiles() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const store = useFileManager();
  const {
    openTabs,
    activeTabPath,
    sidebarWidth,
    sidebarVisible,
    updateContent,
    markSaved,
    closeTab,
    setActiveTab,
    setSidebarWidth,
  } = store;

  // Local dialog state
  const [showNewFile, setShowNewFile] = useState(false);
  const [showNewDir, setShowNewDir] = useState(false);
  const [showRename, setShowRename] = useState<FileEntry | null>(null);
  const [showDelete, setShowDelete] = useState<FileEntry | null>(null);
  const [showQuickOpen, setShowQuickOpen] = useState(false);
  const [showCompress, setShowCompress] = useState<FileEntry | null>(null);
  const [newName, setNewName] = useState("");
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    entry: FileEntry;
  } | null>(null);
  const [contextDir, setContextDir] = useState<string | null>(null);

  // Resize state
  const [isResizing, setIsResizing] = useState(false);
  const resizeRef = useRef<{ startX: number; startWidth: number } | null>(null);

  // Get base path from initial query
  const { data: rootData } = useQuery({
    queryKey: ["files", domainId, undefined],
    queryFn: () => listFiles(domainId, undefined),
    enabled: !!domainId,
  });
  const basePath = rootData?.base;

  // Active tab
  const activeTab = openTabs.find((t) => t.path === activeTabPath);

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: ({ path, content }: { path: string; content: string }) =>
      saveFile(domainId, path, content),
    onSuccess: (_, { path }) => {
      markSaved(path);
      toast.success("File saved");
      queryClient.invalidateQueries({ queryKey: ["files", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to save");
    },
  });

  // Create file mutation
  const createFileMutation = useMutation({
    mutationFn: (fullPath: string) => saveFile(domainId, fullPath, ""),
    onSuccess: () => {
      toast.success("File created");
      setShowNewFile(false);
      setNewName("");
      queryClient.invalidateQueries({ queryKey: ["files", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to create file");
    },
  });

  // Create directory mutation
  const createDirMutation = useMutation({
    mutationFn: (fullPath: string) => createDirectory(domainId, fullPath),
    onSuccess: () => {
      toast.success("Directory created");
      setShowNewDir(false);
      setNewName("");
      queryClient.invalidateQueries({ queryKey: ["files", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to create directory");
    },
  });

  // Rename mutation
  const renameMutation = useMutation({
    mutationFn: ({ oldPath, newPath }: { oldPath: string; newPath: string }) =>
      renameFile(domainId, oldPath, newPath),
    onSuccess: () => {
      toast.success("Renamed successfully");
      setShowRename(null);
      setNewName("");
      queryClient.invalidateQueries({ queryKey: ["files", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to rename");
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: ({ path, recursive }: { path: string; recursive: boolean }) =>
      deleteFile(domainId, path, recursive),
    onSuccess: () => {
      toast.success("Deleted successfully");
      setShowDelete(null);
      queryClient.invalidateQueries({ queryKey: ["files", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to delete");
    },
  });

  // Compress mutation
  const compressMutation = useMutation({
    mutationFn: ({
      sources,
      output,
      format,
    }: {
      sources: string[];
      output: string;
      format: string;
    }) => compressFiles(domainId, sources, output, format),
    onSuccess: () => {
      toast.success("Archive created");
      setShowCompress(null);
      queryClient.invalidateQueries({ queryKey: ["files", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to compress");
    },
  });

  // Extract mutation
  const extractMutation = useMutation({
    mutationFn: ({ archive, dest }: { archive: string; dest: string }) =>
      extractArchive(domainId, archive, dest),
    onSuccess: () => {
      toast.success("Extracted successfully");
      queryClient.invalidateQueries({ queryKey: ["files", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to extract");
    },
  });

  // Save active file
  const handleSave = useCallback(() => {
    if (!activeTab || activeTab.content === activeTab.originalContent) return;
    saveMutation.mutate({ path: activeTab.path, content: activeTab.content });
  }, [activeTab, saveMutation]);

  // Upload handler
  const handleUpload = useCallback(
    async (files: File[]) => {
      const destDir = contextDir ?? basePath;
      if (!destDir) return;
      store.setUploading(true, 0);
      try {
        await uploadFiles(domainId, destDir, files, (progress) => {
          store.setUploading(true, progress);
        });
        toast.success(`Uploaded ${files.length} file(s)`);
        queryClient.invalidateQueries({ queryKey: ["files", domainId] });
      } catch (err) {
        const axErr = err as AxiosError<APIError>;
        toast.error(axErr.response?.data?.error?.message ?? "Upload failed");
      } finally {
        store.setUploading(false, 0);
      }
    },
    [domainId, basePath, contextDir, queryClient, store]
  );

  // Upload button click (via hidden input)
  const fileInputRef = useRef<HTMLInputElement>(null);
  const handleUploadClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  // Context menu handler
  const handleContextMenu = useCallback(
    (e: React.MouseEvent, entry: FileEntry) => {
      e.preventDefault();
      e.stopPropagation();
      setContextMenu({ x: e.clientX, y: e.clientY, entry });
    },
    []
  );

  // Sidebar resize handlers
  const handleResizeStart = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      setIsResizing(true);
      resizeRef.current = { startX: e.clientX, startWidth: sidebarWidth };
    },
    [sidebarWidth]
  );

  useEffect(() => {
    if (!isResizing) return;
    function handleMouseMove(e: MouseEvent) {
      if (!resizeRef.current) return;
      const delta = e.clientX - resizeRef.current.startX;
      setSidebarWidth(resizeRef.current.startWidth + delta);
    }
    function handleMouseUp() {
      setIsResizing(false);
      resizeRef.current = null;
    }
    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isResizing, setSidebarWidth]);

  // Keyboard shortcuts
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const mod = e.metaKey || e.ctrlKey;

      if (mod && e.key === "s") {
        e.preventDefault();
        handleSave();
      } else if (mod && e.key === "p") {
        e.preventDefault();
        setShowQuickOpen(true);
      } else if (mod && e.key === "b") {
        e.preventDefault();
        store.toggleSidebar();
      } else if (mod && e.key === "n") {
        e.preventDefault();
        setShowNewFile(true);
        setNewName("");
      } else if (mod && e.key === "w") {
        e.preventDefault();
        if (activeTabPath) closeTab(activeTabPath);
      } else if (e.ctrlKey && e.key === "Tab") {
        e.preventDefault();
        if (openTabs.length > 1 && activeTabPath) {
          const idx = openTabs.findIndex((t) => t.path === activeTabPath);
          const nextIdx = e.shiftKey
            ? (idx - 1 + openTabs.length) % openTabs.length
            : (idx + 1) % openTabs.length;
          setActiveTab(openTabs[nextIdx].path);
        }
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleSave, activeTabPath, openTabs, closeTab, setActiveTab, store]);

  // Determine what to show in the selected directory for new file/folder
  const selectedDir = (() => {
    if (contextDir) return contextDir;
    if (store.selectedPath) {
      // Check if selected path is a directory by checking expanded dirs or open tabs
      const tab = openTabs.find((t) => t.path === store.selectedPath);
      if (!tab) {
        // Might be a directory
        return store.selectedPath;
      }
      // It's a file — use its parent
      return store.selectedPath.substring(0, store.selectedPath.lastIndexOf("/"));
    }
    return basePath ?? "";
  })();

  // Loading state
  if (!rootData) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  const isImage = activeTab?.content === "__image__";

  return (
    <UploadZone onDrop={handleUpload} disabled={store.isUploading}>
      <div
        className="flex border rounded-lg overflow-hidden bg-background"
        style={{ height: "calc(100vh - 220px)", minHeight: "400px" }}
      >
        {/* Sidebar */}
        {sidebarVisible && (
          <>
            <div
              className="flex flex-col border-r bg-muted/20 shrink-0 overflow-hidden"
              style={{ width: sidebarWidth }}
            >
              <FileToolbar
                onNewFile={() => {
                  setContextDir(selectedDir);
                  setShowNewFile(true);
                  setNewName("");
                }}
                onNewFolder={() => {
                  setContextDir(selectedDir);
                  setShowNewDir(true);
                  setNewName("");
                }}
                onUpload={handleUploadClick}
                onSearch={() => setShowQuickOpen(true)}
              />
              <div className="flex-1 overflow-y-auto overflow-x-hidden">
                <FileTree
                  domainId={domainId}
                  basePath={basePath}
                  onContextMenu={handleContextMenu}
                />
              </div>
            </div>
            {/* Resize handle */}
            <div
              className={`w-1 cursor-col-resize hover:bg-pink-500/30 transition-colors shrink-0 ${
                isResizing ? "bg-pink-500/50" : ""
              }`}
              onMouseDown={handleResizeStart}
            />
          </>
        )}

        {/* Editor area */}
        <div className="flex flex-col flex-1 min-w-0">
          <EditorTabs />
          <div className="flex-1 overflow-hidden">
            {!activeTab && <WelcomePanel />}
            {activeTab?.isLoading && (
              <div className="flex items-center justify-center h-full">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            )}
            {activeTab && !activeTab.isLoading && isImage && (
              <ImagePreview
                domainId={domainId}
                path={activeTab.path}
                name={activeTab.name}
              />
            )}
            {activeTab && !activeTab.isLoading && !isImage && (
              <CodeEditor
                content={activeTab.content}
                language={activeTab.language}
                onChange={(content) => updateContent(activeTab.path, content)}
                onSave={handleSave}
              />
            )}
          </div>
        </div>
      </div>

      {/* Hidden file input for upload button */}
      <input
        ref={fileInputRef}
        type="file"
        multiple
        className="hidden"
        onChange={(e) => {
          if (e.target.files?.length) {
            handleUpload(Array.from(e.target.files));
            e.target.value = "";
          }
        }}
      />

      {/* Context menu */}
      {contextMenu && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          entry={contextMenu.entry}
          onClose={() => setContextMenu(null)}
          onOpen={() => {
            // Opening is handled by tree click
          }}
          onRename={() => {
            setShowRename(contextMenu.entry);
            setNewName(contextMenu.entry.name);
          }}
          onCopyPath={() => {
            navigator.clipboard.writeText(contextMenu.entry.path);
            toast.success("Path copied");
          }}
          onDownload={() => {
            downloadFile(domainId, contextMenu.entry.path).catch(() => {
              toast.error("Download failed");
            });
          }}
          onDelete={() => {
            setShowDelete(contextMenu.entry);
          }}
          onNewFile={() => {
            setContextDir(contextMenu.entry.is_dir ? contextMenu.entry.path : contextMenu.entry.path.substring(0, contextMenu.entry.path.lastIndexOf("/")));
            setShowNewFile(true);
            setNewName("");
          }}
          onNewFolder={() => {
            setContextDir(contextMenu.entry.is_dir ? contextMenu.entry.path : contextMenu.entry.path.substring(0, contextMenu.entry.path.lastIndexOf("/")));
            setShowNewDir(true);
            setNewName("");
          }}
          onCompress={() => {
            setShowCompress(contextMenu.entry);
          }}
          onExtract={() => {
            const dest = contextMenu.entry.path.substring(
              0,
              contextMenu.entry.path.lastIndexOf("/")
            );
            extractMutation.mutate({
              archive: contextMenu.entry.path,
              dest,
            });
          }}
        />
      )}

      {/* Quick Open */}
      <QuickOpen
        open={showQuickOpen}
        onOpenChange={setShowQuickOpen}
        domainId={domainId}
        basePath={basePath}
      />

      {/* New File Dialog */}
      <Dialog open={showNewFile} onOpenChange={setShowNewFile}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New File</DialogTitle>
            <DialogDescription>Create a new empty file</DialogDescription>
          </DialogHeader>
          <Input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="filename.txt"
            autoFocus
            onKeyDown={(e) => {
              if (e.key === "Enter" && newName) {
                createFileMutation.mutate(`${selectedDir}/${newName}`);
              }
            }}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowNewFile(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => createFileMutation.mutate(`${selectedDir}/${newName}`)}
              disabled={!newName || createFileMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {createFileMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* New Directory Dialog */}
      <Dialog open={showNewDir} onOpenChange={setShowNewDir}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New Folder</DialogTitle>
            <DialogDescription>Create a new directory</DialogDescription>
          </DialogHeader>
          <Input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="folder-name"
            autoFocus
            onKeyDown={(e) => {
              if (e.key === "Enter" && newName) {
                createDirMutation.mutate(`${selectedDir}/${newName}`);
              }
            }}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowNewDir(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => createDirMutation.mutate(`${selectedDir}/${newName}`)}
              disabled={!newName || createDirMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {createDirMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Rename Dialog */}
      <Dialog open={!!showRename} onOpenChange={() => setShowRename(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rename</DialogTitle>
            <DialogDescription>
              Rename {showRename?.is_dir ? "folder" : "file"} "{showRename?.name}"
            </DialogDescription>
          </DialogHeader>
          <Input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            autoFocus
            onKeyDown={(e) => {
              if (e.key === "Enter" && newName && newName !== showRename?.name) {
                const dir = showRename!.path.substring(0, showRename!.path.lastIndexOf("/"));
                renameMutation.mutate({ oldPath: showRename!.path, newPath: `${dir}/${newName}` });
              }
            }}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowRename(null)}>
              Cancel
            </Button>
            <Button
              onClick={() => {
                const dir = showRename!.path.substring(0, showRename!.path.lastIndexOf("/"));
                renameMutation.mutate({ oldPath: showRename!.path, newPath: `${dir}/${newName}` });
              }}
              disabled={!newName || newName === showRename?.name || renameMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {renameMutation.isPending ? "Renaming..." : "Rename"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Compress Dialog */}
      {showCompress && (
        <CompressDialog
          open={!!showCompress}
          onOpenChange={() => setShowCompress(null)}
          sourceName={showCompress.name}
          loading={compressMutation.isPending}
          onCompress={(outputName, format) => {
            const dir = showCompress.path.substring(
              0,
              showCompress.path.lastIndexOf("/")
            );
            compressMutation.mutate({
              sources: [showCompress.path],
              output: `${dir}/${outputName}`,
              format,
            });
          }}
        />
      )}

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!showDelete}
        onOpenChange={() => setShowDelete(null)}
        title={`Delete ${showDelete?.is_dir ? "Folder" : "File"}`}
        description={`Are you sure you want to delete "${showDelete?.name}"?${showDelete?.is_dir ? " This will delete all contents." : ""}`}
        confirmText="Delete"
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() =>
          deleteMutation.mutate({
            path: showDelete!.path,
            recursive: showDelete!.is_dir,
          })
        }
      />

      {/* Upload progress toast */}
      {store.isUploading && (
        <div className="fixed bottom-4 right-4 z-50 bg-popover border rounded-lg shadow-lg p-3 flex items-center gap-3">
          <Loader2 className="h-4 w-4 animate-spin text-pink-500" />
          <span className="text-sm">Uploading... {store.uploadProgress}%</span>
        </div>
      )}
    </UploadZone>
  );
}
