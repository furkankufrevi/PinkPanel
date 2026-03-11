import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface CompressDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sourceName: string;
  onCompress: (outputName: string, format: string) => void;
  loading?: boolean;
}

const formats = [
  { value: "zip", label: "ZIP (.zip)" },
  { value: "tar.gz", label: "TAR.GZ (.tar.gz)" },
  { value: "tar.bz2", label: "TAR.BZ2 (.tar.bz2)" },
];

export function CompressDialog({
  open,
  onOpenChange,
  sourceName,
  onCompress,
  loading,
}: CompressDialogProps) {
  const [format, setFormat] = useState("zip");
  const [outputName, setOutputName] = useState(sourceName);

  function getExtension(fmt: string) {
    return fmt === "zip" ? ".zip" : fmt === "tar.gz" ? ".tar.gz" : ".tar.bz2";
  }

  function handleSubmit() {
    let name = outputName;
    const ext = getExtension(format);
    if (!name.endsWith(ext)) {
      name += ext;
    }
    onCompress(name, format);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Compress</DialogTitle>
          <DialogDescription>
            Create an archive from "{sourceName}"
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Output filename</Label>
            <Input
              value={outputName}
              onChange={(e) => setOutputName(e.target.value)}
              placeholder="archive"
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label>Format</Label>
            <div className="flex gap-2">
              {formats.map((f) => (
                <button
                  key={f.value}
                  className={`px-3 py-1.5 rounded-md text-sm border transition-colors ${
                    format === f.value
                      ? "border-pink-500 bg-pink-500/10 text-pink-500"
                      : "border-border hover:border-foreground/30"
                  }`}
                  onClick={() => setFormat(f.value)}
                >
                  {f.label}
                </button>
              ))}
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!outputName || loading}
            className="bg-pink-500 hover:bg-pink-600"
          >
            {loading ? "Compressing..." : "Compress"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
