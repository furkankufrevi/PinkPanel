import { useFileManager, type OpenTab } from "@/stores/file-manager";
import {
  FileCode,
  FileText,
  File,
  Image,
  X,
} from "lucide-react";

function getTabIcon(name: string) {
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  if (["jpg", "jpeg", "png", "gif", "svg", "webp", "ico"].includes(ext))
    return <Image className="h-3.5 w-3.5 text-purple-400" />;
  if (["php", "js", "ts", "jsx", "tsx", "py", "rb", "go", "rs", "css", "scss", "json", "xml", "yaml", "yml", "sh"].includes(ext))
    return <FileCode className="h-3.5 w-3.5 text-blue-400" />;
  if (["txt", "md", "log", "conf", "cfg", "ini", "env", "htaccess"].includes(ext))
    return <FileText className="h-3.5 w-3.5 text-muted-foreground" />;
  return <File className="h-3.5 w-3.5 text-muted-foreground" />;
}

export function EditorTabs() {
  const { openTabs, activeTabPath, setActiveTab, closeTab } = useFileManager();

  if (openTabs.length === 0) return null;

  function handleMouseDown(e: React.MouseEvent, tab: OpenTab) {
    // Middle click to close
    if (e.button === 1) {
      e.preventDefault();
      closeTab(tab.path);
    }
  }

  return (
    <div className="flex items-center border-b bg-muted/30 overflow-x-auto scrollbar-none">
      {openTabs.map((tab) => {
        const isActive = tab.path === activeTabPath;
        const isDirty = tab.content !== tab.originalContent;
        return (
          <button
            key={tab.path}
            className={`group flex items-center gap-1.5 px-3 py-1.5 text-xs border-r border-border min-w-0 shrink-0 transition-colors ${
              isActive
                ? "bg-background text-foreground"
                : "text-muted-foreground hover:bg-muted/50 hover:text-foreground"
            }`}
            onClick={() => setActiveTab(tab.path)}
            onMouseDown={(e) => handleMouseDown(e, tab)}
            title={tab.path}
          >
            {getTabIcon(tab.name)}
            <span className="truncate max-w-[120px]">{tab.name}</span>
            {isDirty && (
              <span className="h-2 w-2 rounded-full bg-pink-500 shrink-0" />
            )}
            <span
              className={`ml-1 rounded p-0.5 shrink-0 hover:bg-muted-foreground/20 ${
                isActive ? "opacity-60 hover:opacity-100" : "opacity-0 group-hover:opacity-60 hover:!opacity-100"
              }`}
              onClick={(e) => {
                e.stopPropagation();
                closeTab(tab.path);
              }}
            >
              <X className="h-3 w-3" />
            </span>
          </button>
        );
      })}
    </div>
  );
}
