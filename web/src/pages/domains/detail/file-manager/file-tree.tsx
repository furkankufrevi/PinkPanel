import { useQuery } from "@tanstack/react-query";
import { listFiles } from "@/api/files";
import { FileTreeNode } from "./file-tree-node";
import type { FileEntry } from "@/types/files";
import { Loader2 } from "lucide-react";

interface FileTreeProps {
  domainId: number;
  basePath?: string;
  onContextMenu: (e: React.MouseEvent, entry: FileEntry) => void;
}

export function FileTree({ domainId, basePath, onContextMenu }: FileTreeProps) {
  const { data, isLoading } = useQuery({
    queryKey: ["files", domainId, basePath],
    queryFn: () => listFiles(domainId, basePath),
    enabled: !!domainId,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  const entries = (data?.data ?? []) as FileEntry[];
  const dirs = entries.filter((e) => e.is_dir).sort((a, b) => a.name.localeCompare(b.name));
  const files = entries.filter((e) => !e.is_dir).sort((a, b) => a.name.localeCompare(b.name));
  const sorted = [...dirs, ...files];

  if (sorted.length === 0) {
    return (
      <div className="px-3 py-4 text-xs text-muted-foreground">
        Empty directory
      </div>
    );
  }

  return (
    <div className="py-1 select-none">
      {sorted.map((entry) => (
        <FileTreeNode
          key={entry.path}
          entry={entry}
          domainId={domainId}
          depth={0}
          onContextMenu={onContextMenu}
        />
      ))}
    </div>
  );
}
