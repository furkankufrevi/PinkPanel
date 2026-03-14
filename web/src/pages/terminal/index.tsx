import { useEffect, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { getAccessToken } from "@/api/client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { TerminalIcon, Wifi, WifiOff } from "lucide-react";
import "@xterm/xterm/css/xterm.css";

export function TerminalPage() {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function connect() {
    if (!containerRef.current) return;

    setError(null);

    // Clean up previous
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    if (termRef.current) {
      termRef.current.dispose();
      termRef.current = null;
    }

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "'Geist Mono', 'Fira Code', 'Cascadia Code', Menlo, monospace",
      theme: {
        background: "#09090b",
        foreground: "#fafafa",
        cursor: "#ec4899",
        selectionBackground: "#ec489944",
        black: "#09090b",
        red: "#ef4444",
        green: "#22c55e",
        yellow: "#eab308",
        blue: "#3b82f6",
        magenta: "#ec4899",
        cyan: "#06b6d4",
        white: "#fafafa",
        brightBlack: "#71717a",
        brightRed: "#f87171",
        brightGreen: "#4ade80",
        brightYellow: "#facc15",
        brightBlue: "#60a5fa",
        brightMagenta: "#f472b6",
        brightCyan: "#22d3ee",
        brightWhite: "#ffffff",
      },
    });

    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();
    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);

    term.open(containerRef.current);
    fitAddon.fit();

    termRef.current = term;
    fitRef.current = fitAddon;

    // Connect WebSocket
    const token = getAccessToken();
    if (!token) {
      setError("Not authenticated");
      return;
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/api/terminal/ws?token=${encodeURIComponent(token)}`;
    const ws = new WebSocket(wsUrl);
    ws.binaryType = "arraybuffer";
    wsRef.current = ws;

    ws.onopen = () => {
      setConnected(true);
      // Send initial config
      ws.send(JSON.stringify({ cols: term.cols, rows: term.rows }));
    };

    ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(event.data));
      } else {
        // Text message (e.g. error from server)
        try {
          const data = JSON.parse(event.data);
          if (data.type === "error") {
            setError(data.message);
            return;
          }
        } catch {
          // Not JSON, write as text
        }
        term.write(event.data);
      }
    };

    ws.onclose = () => {
      setConnected(false);
      term.write("\r\n\x1b[90m--- Session ended ---\x1b[0m\r\n");
    };

    ws.onerror = () => {
      setError("Connection failed");
      ws.close();
    };

    // Terminal input → WebSocket
    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(new TextEncoder().encode(data));
      }
    });

    // Terminal resize → WebSocket
    term.onResize(({ cols, rows }) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "resize", cols, rows }));
      }
    });

    // Focus the terminal
    term.focus();
  }

  useEffect(() => {
    connect();

    // Handle window resize
    const handleResize = () => {
      if (fitRef.current) {
        fitRef.current.fit();
      }
    };
    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
      if (wsRef.current) wsRef.current.close();
      if (termRef.current) termRef.current.dispose();
    };
  }, []);

  return (
    <div className="flex flex-col h-[calc(100vh-7rem)]">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <TerminalIcon className="h-5 w-5 text-pink-500" />
          <h1 className="text-2xl font-bold">Terminal</h1>
        </div>
        <div className="flex items-center gap-2">
          <Badge
            variant="outline"
            className={
              connected
                ? "bg-green-500/10 text-green-500 border-green-500/20"
                : "bg-muted text-muted-foreground"
            }
          >
            {connected ? (
              <Wifi className="h-3 w-3 mr-1" />
            ) : (
              <WifiOff className="h-3 w-3 mr-1" />
            )}
            {connected ? "Connected" : "Disconnected"}
          </Badge>
          {!connected && !error && (
            <Button size="sm" variant="outline" onClick={connect}>
              Reconnect
            </Button>
          )}
        </div>
      </div>

      {error && (
        <div className="mb-3 rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive flex items-center justify-between">
          <span>{error}</span>
          <Button size="sm" variant="outline" onClick={connect}>
            Retry
          </Button>
        </div>
      )}

      <div
        ref={containerRef}
        className="flex-1 rounded-lg border border-border overflow-hidden bg-[#09090b] p-1"
      />
    </div>
  );
}
