import { Button } from "@/components/ui/button";
import { useFileManager } from "@/stores/file-manager";
import {
  Plus,
  FolderPlus,
  Upload,
  Search,
  PanelLeftClose,
  PanelLeft,
} from "lucide-react";

interface FileToolbarProps {
  onNewFile: () => void;
  onNewFolder: () => void;
  onUpload: () => void;
  onSearch: () => void;
}

export function FileToolbar({ onNewFile, onNewFolder, onUpload, onSearch }: FileToolbarProps) {
  const { sidebarVisible, toggleSidebar } = useFileManager();

  return (
    <div className="flex items-center justify-between px-2 py-1.5 border-b bg-muted/30">
      <div className="flex items-center gap-1">
        <Button
          size="icon"
          variant="ghost"
          className="h-7 w-7"
          onClick={toggleSidebar}
          title={sidebarVisible ? "Hide sidebar (Ctrl+B)" : "Show sidebar (Ctrl+B)"}
        >
          {sidebarVisible ? <PanelLeftClose className="h-4 w-4" /> : <PanelLeft className="h-4 w-4" />}
        </Button>
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider ml-1">
          Files
        </span>
      </div>
      <div className="flex items-center gap-0.5">
        <Button size="icon" variant="ghost" className="h-7 w-7" onClick={onSearch} title="Search (Ctrl+P)">
          <Search className="h-4 w-4" />
        </Button>
        <Button size="icon" variant="ghost" className="h-7 w-7" onClick={onNewFile} title="New File (Ctrl+N)">
          <Plus className="h-4 w-4" />
        </Button>
        <Button size="icon" variant="ghost" className="h-7 w-7" onClick={onNewFolder} title="New Folder">
          <FolderPlus className="h-4 w-4" />
        </Button>
        <Button size="icon" variant="ghost" className="h-7 w-7" onClick={onUpload} title="Upload Files">
          <Upload className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
