import {
  Keyboard,
  FileCode,
} from "lucide-react";

const shortcuts = [
  { keys: "Ctrl+S", action: "Save current file" },
  { keys: "Ctrl+P", action: "Quick search / open" },
  { keys: "Ctrl+B", action: "Toggle sidebar" },
  { keys: "Ctrl+N", action: "New file" },
  { keys: "Ctrl+W", action: "Close current tab" },
  { keys: "Ctrl+Tab", action: "Next tab" },
  { keys: "Ctrl+Shift+Tab", action: "Previous tab" },
];

export function WelcomePanel() {
  return (
    <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-6 p-8">
      <FileCode className="h-16 w-16 opacity-20" />
      <div className="text-center space-y-1">
        <h3 className="text-lg font-medium text-foreground">No file open</h3>
        <p className="text-sm">Select a file from the tree to start editing</p>
      </div>
      <div className="bg-muted/50 rounded-lg p-4 w-full max-w-xs">
        <div className="flex items-center gap-2 mb-3 text-foreground">
          <Keyboard className="h-4 w-4" />
          <span className="text-sm font-medium">Keyboard Shortcuts</span>
        </div>
        <div className="space-y-1.5">
          {shortcuts.map((s) => (
            <div key={s.keys} className="flex items-center justify-between text-xs">
              <span>{s.action}</span>
              <kbd className="px-1.5 py-0.5 rounded bg-background border text-[10px] font-mono">
                {s.keys}
              </kbd>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
