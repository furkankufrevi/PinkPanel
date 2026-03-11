import { useEffect, useRef } from "react";
import type { FileEntry } from "@/types/files";
import {
  FileText,
  FolderPlus,
  Plus,
  Pencil,
  Copy,
  Download,
  Trash2,
  Archive,
  PackageOpen,
} from "lucide-react";

interface ContextMenuProps {
  x: number;
  y: number;
  entry: FileEntry;
  onClose: () => void;
  onOpen: () => void;
  onRename: () => void;
  onCopyPath: () => void;
  onDownload: () => void;
  onDelete: () => void;
  onNewFile: () => void;
  onNewFolder: () => void;
  onCompress: () => void;
  onExtract: () => void;
}

function isArchive(name: string): boolean {
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  return ["zip", "tar", "gz", "tgz", "bz2", "tar.gz", "tar.bz2"].includes(ext) ||
    name.endsWith(".tar.gz") || name.endsWith(".tar.bz2");
}

export function ContextMenu({
  x,
  y,
  entry,
  onClose,
  onOpen,
  onRename,
  onCopyPath,
  onDownload,
  onDelete,
  onNewFile,
  onNewFolder,
  onCompress,
  onExtract,
}: ContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose();
      }
    }
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("mousedown", handleClick);
    document.addEventListener("keydown", handleKey);
    return () => {
      document.removeEventListener("mousedown", handleClick);
      document.removeEventListener("keydown", handleKey);
    };
  }, [onClose]);

  // Adjust position to keep menu in viewport
  useEffect(() => {
    if (menuRef.current) {
      const rect = menuRef.current.getBoundingClientRect();
      const vw = window.innerWidth;
      const vh = window.innerHeight;
      if (rect.right > vw) menuRef.current.style.left = `${x - rect.width}px`;
      if (rect.bottom > vh) menuRef.current.style.top = `${y - rect.height}px`;
    }
  }, [x, y]);

  const items: Array<{
    icon: React.ReactNode;
    label: string;
    onClick: () => void;
    destructive?: boolean;
    separator?: boolean;
  }> = [];

  if (entry.is_dir) {
    items.push({ icon: <Plus className="h-4 w-4" />, label: "New File", onClick: onNewFile });
    items.push({ icon: <FolderPlus className="h-4 w-4" />, label: "New Folder", onClick: onNewFolder });
    items.push({ icon: <span />, label: "", onClick: () => {}, separator: true });
  } else {
    items.push({ icon: <FileText className="h-4 w-4" />, label: "Open", onClick: onOpen });
  }

  items.push({ icon: <Pencil className="h-4 w-4" />, label: "Rename", onClick: onRename });
  items.push({ icon: <Copy className="h-4 w-4" />, label: "Copy Path", onClick: onCopyPath });
  items.push({ icon: <Download className="h-4 w-4" />, label: entry.is_dir ? "Download as ZIP" : "Download", onClick: onDownload });

  if (entry.is_dir || !isArchive(entry.name)) {
    items.push({ icon: <Archive className="h-4 w-4" />, label: "Compress", onClick: onCompress });
  }

  if (isArchive(entry.name)) {
    items.push({ icon: <PackageOpen className="h-4 w-4" />, label: "Extract Here", onClick: onExtract });
  }

  items.push({ icon: <span />, label: "", onClick: () => {}, separator: true });
  items.push({ icon: <Trash2 className="h-4 w-4" />, label: "Delete", onClick: onDelete, destructive: true });

  return (
    <div
      ref={menuRef}
      className="fixed z-[100] min-w-[180px] rounded-lg border bg-popover p-1 shadow-lg animate-in fade-in-0 zoom-in-95"
      style={{ left: x, top: y }}
    >
      {items.map((item, i) => {
        if (item.separator) {
          return <div key={i} className="my-1 h-px bg-border" />;
        }
        return (
          <button
            key={i}
            className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
              item.destructive
                ? "text-destructive hover:bg-destructive/10"
                : "hover:bg-accent hover:text-accent-foreground"
            }`}
            onClick={() => {
              item.onClick();
              onClose();
            }}
          >
            {item.icon}
            {item.label}
          </button>
        );
      })}
    </div>
  );
}
