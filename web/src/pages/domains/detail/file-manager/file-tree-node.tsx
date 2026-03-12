import { useQuery } from "@tanstack/react-query";
import { routedListFiles, routedReadFile } from "@/api/files";
import { useFileManager } from "@/stores/file-manager";
import type { FileEntry } from "@/types/files";
import {
  Folder,
  FolderOpen,
  FileCode,
  FileText,
  File,
  Image,
  Archive,
  ChevronRight,
  ChevronDown,
} from "lucide-react";

function getFileIcon(entry: FileEntry, isExpanded: boolean) {
  if (entry.is_dir) {
    return isExpanded
      ? <FolderOpen className="h-4 w-4 text-blue-500 shrink-0" />
      : <Folder className="h-4 w-4 text-blue-500 shrink-0" />;
  }
  const ext = entry.name.split(".").pop()?.toLowerCase() ?? "";
  if (["jpg", "jpeg", "png", "gif", "svg", "webp", "ico"].includes(ext))
    return <Image className="h-4 w-4 text-purple-500 shrink-0" />;
  if (["zip", "tar", "gz", "tgz", "bz2", "rar", "7z"].includes(ext))
    return <Archive className="h-4 w-4 text-amber-500 shrink-0" />;
  if (["php", "js", "ts", "jsx", "tsx", "py", "rb", "go", "rs", "css", "scss", "json", "xml", "yaml", "yml", "toml", "sh", "bash"].includes(ext))
    return <FileCode className="h-4 w-4 text-green-500 shrink-0" />;
  if (["txt", "md", "log", "conf", "cfg", "ini", "env", "htaccess"].includes(ext))
    return <FileText className="h-4 w-4 text-muted-foreground shrink-0" />;
  return <File className="h-4 w-4 text-muted-foreground shrink-0" />;
}

interface FileTreeNodeProps {
  entry: FileEntry;
  domainId: number;
  depth: number;
  onContextMenu: (e: React.MouseEvent, entry: FileEntry) => void;
}

export function FileTreeNode({ entry, domainId, depth, onContextMenu }: FileTreeNodeProps) {
  const { expandedDirs, selectedPath, toggleDir, openFile, setTabLoading, setSelectedPath } = useFileManager();
  const isExpanded = expandedDirs.has(entry.path);
  const isSelected = selectedPath === entry.path;

  const { data: children } = useQuery({
    queryKey: ["files", domainId, entry.path],
    queryFn: () => routedListFiles(domainId, entry.path),
    enabled: entry.is_dir && isExpanded,
  });

  async function handleClick() {
    if (entry.is_dir) {
      toggleDir(entry.path);
    } else {
      setSelectedPath(entry.path);
      // Check if the file is an image
      const ext = entry.name.split(".").pop()?.toLowerCase() ?? "";
      const imageExts = ["jpg", "jpeg", "png", "gif", "svg", "webp", "ico"];
      if (imageExts.includes(ext)) {
        openFile(entry.path, entry.name, `__image__`);
        return;
      }
      openFile(entry.path, entry.name);
      setTabLoading(entry.path, true);
      try {
        const result = await routedReadFile(domainId, entry.path);
        // Update the tab content
        const store = useFileManager.getState();
        const tabs = store.openTabs.map((t) =>
          t.path === entry.path
            ? { ...t, content: result.content, originalContent: result.content, isLoading: false }
            : t
        );
        useFileManager.setState({ openTabs: tabs });
      } catch {
        useFileManager.getState().closeTab(entry.path);
      }
    }
  }

  const childEntries = (children?.data ?? []) as FileEntry[];
  const dirs = childEntries.filter((e) => e.is_dir).sort((a, b) => a.name.localeCompare(b.name));
  const files = childEntries.filter((e) => !e.is_dir).sort((a, b) => a.name.localeCompare(b.name));
  const sorted = [...dirs, ...files];

  return (
    <div>
      <button
        className={`flex items-center w-full text-left py-0.5 pr-2 text-sm hover:bg-muted/50 transition-colors ${
          isSelected ? "bg-pink-500/10 text-pink-500" : "text-foreground"
        }`}
        style={{ paddingLeft: `${depth * 16 + 4}px` }}
        onClick={handleClick}
        onContextMenu={(e) => onContextMenu(e, entry)}
      >
        {entry.is_dir ? (
          <span className="shrink-0 w-4 h-4 flex items-center justify-center mr-0.5">
            {isExpanded ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
          </span>
        ) : (
          <span className="w-4 mr-0.5 shrink-0" />
        )}
        {getFileIcon(entry, isExpanded)}
        <span className="ml-1.5 truncate">{entry.name}</span>
      </button>
      {entry.is_dir && isExpanded && (
        <div>
          {sorted.map((child) => (
            <FileTreeNode
              key={child.path}
              entry={child}
              domainId={domainId}
              depth={depth + 1}
              onContextMenu={onContextMenu}
            />
          ))}
        </div>
      )}
    </div>
  );
}
