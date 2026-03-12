import { useState, useEffect, useRef, useCallback } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { routedSearchFiles, routedReadFile } from "@/api/files";
import { useFileManager } from "@/stores/file-manager";
import type { SearchResult } from "@/types/files";
import { FileCode, Search, Loader2 } from "lucide-react";

interface QuickOpenProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  domainId: number;
  basePath?: string;
}

export function QuickOpen({ open, onOpenChange, domainId, basePath }: QuickOpenProps) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const { openFile, setTabLoading } = useFileManager();

  useEffect(() => {
    if (!open) {
      setQuery("");
      setResults([]);
      setSelectedIndex(0);
    }
  }, [open]);

  const doSearch = useCallback(
    async (q: string) => {
      if (!q.trim()) {
        setResults([]);
        return;
      }
      setIsSearching(true);
      try {
        const res = await routedSearchFiles(domainId, q, basePath);
        setResults(res);
        setSelectedIndex(0);
      } catch {
        setResults([]);
      } finally {
        setIsSearching(false);
      }
    },
    [domainId, basePath]
  );

  function handleQueryChange(value: string) {
    setQuery(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => doSearch(value), 300);
  }

  async function openResult(result: SearchResult) {
    const name = result.path.split("/").pop() || result.path;
    openFile(result.path, name);
    setTabLoading(result.path, true);
    onOpenChange(false);
    try {
      const data = await routedReadFile(domainId, result.path);
      const store = useFileManager.getState();
      const tabs = store.openTabs.map((t) =>
        t.path === result.path
          ? { ...t, content: data.content, originalContent: data.content, isLoading: false }
          : t
      );
      useFileManager.setState({ openTabs: tabs });
    } catch {
      useFileManager.getState().closeTab(result.path);
    }
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelectedIndex((i) => Math.min(i + 1, results.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIndex((i) => Math.max(i - 1, 0));
    } else if (e.key === "Enter" && results[selectedIndex]) {
      e.preventDefault();
      openResult(results[selectedIndex]);
    }
  }

  // Strip basePath prefix for display
  function displayPath(path: string) {
    if (basePath && path.startsWith(basePath)) {
      return path.slice(basePath.length + 1);
    }
    return path;
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg p-0 gap-0 overflow-hidden">
        <DialogHeader className="p-4 pb-0">
          <DialogTitle className="text-sm font-medium">Search Files</DialogTitle>
        </DialogHeader>
        <div className="p-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              value={query}
              onChange={(e) => handleQueryChange(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Search file contents..."
              className="pl-9"
              autoFocus
            />
          </div>
        </div>
        <div className="max-h-[300px] overflow-y-auto border-t">
          {isSearching && (
            <div className="flex items-center justify-center py-6">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          )}
          {!isSearching && query && results.length === 0 && (
            <div className="py-6 text-center text-sm text-muted-foreground">
              No results found
            </div>
          )}
          {!isSearching &&
            results.map((result, i) => (
              <button
                key={`${result.path}:${result.line}`}
                className={`flex items-start gap-3 w-full px-4 py-2 text-left text-sm transition-colors ${
                  i === selectedIndex ? "bg-accent text-accent-foreground" : "hover:bg-muted/50"
                }`}
                onClick={() => openResult(result)}
                onMouseEnter={() => setSelectedIndex(i)}
              >
                <FileCode className="h-4 w-4 mt-0.5 shrink-0 text-muted-foreground" />
                <div className="min-w-0 flex-1">
                  <div className="font-mono text-xs truncate">{displayPath(result.path)}</div>
                  <div className="text-xs text-muted-foreground mt-0.5 truncate">
                    <span className="text-pink-500">L{result.line}:</span> {result.snippet}
                  </div>
                </div>
              </button>
            ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}
