import { useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
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
  readFile,
  saveFile,
  deleteFile,
  renameFile,
  createDirectory,
} from "@/api/files";
import type { FileEntry } from "@/types/files";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import {
  Folder,
  File,
  FileText,
  FileCode,
  Image,
  Archive,
  ChevronRight,
  ArrowUp,
  Plus,
  FolderPlus,
  Pencil,
  Trash2,
  Save,
  X,
} from "lucide-react";

function formatSize(bytes: number): string {
  if (bytes === 0) return "—";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

function getFileIcon(entry: FileEntry) {
  if (entry.is_dir) return <Folder className="h-4 w-4 text-blue-500" />;
  const ext = entry.name.split(".").pop()?.toLowerCase() ?? "";
  if (["jpg", "jpeg", "png", "gif", "svg", "webp", "ico"].includes(ext))
    return <Image className="h-4 w-4 text-purple-500" />;
  if (["zip", "tar", "gz", "tgz", "bz2", "rar", "7z"].includes(ext))
    return <Archive className="h-4 w-4 text-amber-500" />;
  if (["php", "js", "ts", "jsx", "tsx", "py", "rb", "go", "rs", "css", "scss", "json", "xml", "yaml", "yml", "toml", "sh", "bash"].includes(ext))
    return <FileCode className="h-4 w-4 text-green-500" />;
  if (["txt", "md", "log", "conf", "cfg", "ini", "env", "htaccess"].includes(ext))
    return <FileText className="h-4 w-4 text-muted-foreground" />;
  return <File className="h-4 w-4 text-muted-foreground" />;
}

export function DomainFiles() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const [currentPath, setCurrentPath] = useState<string | undefined>(undefined);
  const [editingFile, setEditingFile] = useState<string | null>(null);
  const [editContent, setEditContent] = useState("");
  const [showNewFile, setShowNewFile] = useState(false);
  const [showNewDir, setShowNewDir] = useState(false);
  const [showRename, setShowRename] = useState<FileEntry | null>(null);
  const [showDelete, setShowDelete] = useState<FileEntry | null>(null);
  const [newName, setNewName] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["files", domainId, currentPath],
    queryFn: () => listFiles(domainId, currentPath),
    enabled: !!domainId,
  });

  const readMutation = useMutation({
    mutationFn: (path: string) => readFile(domainId, path),
    onSuccess: (result, path) => {
      setEditingFile(path);
      setEditContent(result.content);
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to read file");
    },
  });

  const saveMutation = useMutation({
    mutationFn: () => saveFile(domainId, editingFile!, editContent),
    onSuccess: () => {
      toast.success("File saved");
      setEditingFile(null);
      queryClient.invalidateQueries({ queryKey: ["files", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to save file");
    },
  });

  const createFileMutation = useMutation({
    mutationFn: () => {
      const path = `${data?.path ?? ""}/${newName}`;
      return saveFile(domainId, path, "");
    },
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

  const createDirMutation = useMutation({
    mutationFn: () => {
      const path = `${data?.path ?? ""}/${newName}`;
      return createDirectory(domainId, path);
    },
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

  const renameMutation = useMutation({
    mutationFn: () => {
      const dir = showRename!.path.substring(0, showRename!.path.lastIndexOf("/"));
      const newPath = `${dir}/${newName}`;
      return renameFile(domainId, showRename!.path, newPath);
    },
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

  const deleteMutation = useMutation({
    mutationFn: () => deleteFile(domainId, showDelete!.path, showDelete!.is_dir),
    onSuccess: () => {
      toast.success("Deleted successfully");
      setShowDelete(null);
      queryClient.invalidateQueries({ queryKey: ["files", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to delete");
    },
  });

  function navigateTo(path: string) {
    setCurrentPath(path);
    setEditingFile(null);
  }

  function navigateUp() {
    if (!data?.path || data.path === data.base) return;
    const parent = data.path.substring(0, data.path.lastIndexOf("/"));
    if (parent.length >= (data.base?.length ?? 0)) {
      setCurrentPath(parent);
      setEditingFile(null);
    }
  }

  function handleEntryClick(entry: FileEntry) {
    if (entry.is_dir) {
      navigateTo(entry.path);
    } else {
      readMutation.mutate(entry.path);
    }
  }

  // Breadcrumb
  const breadcrumbs = (() => {
    if (!data?.path || !data?.base) return [];
    const relative = data.path.substring(data.base.length);
    if (!relative) return [{ name: "/", path: data.base }];
    const parts = relative.split("/").filter(Boolean);
    const crumbs = [{ name: "/", path: data.base }];
    let acc = data.base;
    for (const part of parts) {
      acc += "/" + part;
      crumbs.push({ name: part, path: acc });
    }
    return crumbs;
  })();

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  // File editor view
  if (editingFile) {
    const fileName = editingFile.split("/").pop();
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <FileCode className="h-5 w-5 text-muted-foreground" />
            <span className="font-mono text-sm">{fileName}</span>
            <Badge variant="outline" className="text-xs">
              {editingFile}
            </Badge>
          </div>
          <div className="flex gap-2">
            <Button
              size="sm"
              onClick={() => saveMutation.mutate()}
              disabled={saveMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              <Save className="h-4 w-4 mr-1" />
              {saveMutation.isPending ? "Saving..." : "Save"}
            </Button>
            <Button size="sm" variant="outline" onClick={() => setEditingFile(null)}>
              <X className="h-4 w-4 mr-1" />
              Close
            </Button>
          </div>
        </div>
        <Textarea
          value={editContent}
          onChange={(e) => setEditContent(e.target.value)}
          className="font-mono text-xs min-h-[500px]"
          rows={30}
        />
      </div>
    );
  }

  const entries = (data?.data ?? []) as FileEntry[];
  const dirs = entries.filter((e) => e.is_dir).sort((a, b) => a.name.localeCompare(b.name));
  const files = entries.filter((e) => !e.is_dir).sort((a, b) => a.name.localeCompare(b.name));
  const sorted = [...dirs, ...files];

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1 text-sm">
          {data?.path !== data?.base && (
            <Button size="sm" variant="ghost" onClick={navigateUp}>
              <ArrowUp className="h-4 w-4" />
            </Button>
          )}
          {breadcrumbs.map((crumb, i) => (
            <span key={crumb.path} className="flex items-center">
              {i > 0 && <ChevronRight className="h-3 w-3 text-muted-foreground mx-1" />}
              <button
                onClick={() => navigateTo(crumb.path)}
                className="hover:text-pink-500 transition-colors"
              >
                {crumb.name}
              </button>
            </span>
          ))}
        </div>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" onClick={() => { setShowNewFile(true); setNewName(""); }}>
            <Plus className="h-4 w-4 mr-1" />
            New File
          </Button>
          <Button size="sm" variant="outline" onClick={() => { setShowNewDir(true); setNewName(""); }}>
            <FolderPlus className="h-4 w-4 mr-1" />
            New Folder
          </Button>
        </div>
      </div>

      {/* File List */}
      <Card>
        <CardHeader className="py-3">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            {sorted.length} items
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <div className="divide-y">
            {sorted.length === 0 && (
              <div className="p-8 text-center text-muted-foreground text-sm">
                This directory is empty
              </div>
            )}
            {sorted.map((entry) => (
              <div
                key={entry.path}
                className="flex items-center justify-between px-4 py-2 hover:bg-muted/50 group"
              >
                <button
                  className="flex items-center gap-3 flex-1 text-left"
                  onClick={() => handleEntryClick(entry)}
                >
                  {getFileIcon(entry)}
                  <span className="text-sm">{entry.name}</span>
                </button>
                <div className="flex items-center gap-4">
                  <span className="text-xs text-muted-foreground w-16 text-right">
                    {entry.is_dir ? "—" : formatSize(entry.size)}
                  </span>
                  <span className="text-xs text-muted-foreground font-mono w-20">
                    {entry.permissions}
                  </span>
                  <span className="text-xs text-muted-foreground w-32">
                    {new Date(entry.mod_time).toLocaleString()}
                  </span>
                  <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-7 w-7"
                      onClick={(e) => {
                        e.stopPropagation();
                        setShowRename(entry);
                        setNewName(entry.name);
                      }}
                    >
                      <Pencil className="h-3 w-3" />
                    </Button>
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-7 w-7 text-destructive"
                      onClick={(e) => {
                        e.stopPropagation();
                        setShowDelete(entry);
                      }}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

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
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowNewFile(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => createFileMutation.mutate()}
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
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowNewDir(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => createDirMutation.mutate()}
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
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowRename(null)}>
              Cancel
            </Button>
            <Button
              onClick={() => renameMutation.mutate()}
              disabled={!newName || newName === showRename?.name || renameMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {renameMutation.isPending ? "Renaming..." : "Rename"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!showDelete}
        onOpenChange={() => setShowDelete(null)}
        title={`Delete ${showDelete?.is_dir ? "Folder" : "File"}`}
        description={`Are you sure you want to delete "${showDelete?.name}"?${showDelete?.is_dir ? " This will delete all contents." : ""}`}
        confirmText="Delete"
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
    </div>
  );
}
