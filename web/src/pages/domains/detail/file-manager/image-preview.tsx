import { getDownloadUrl } from "@/api/files";
import { getAccessToken } from "@/api/client";
import { Image as ImageIcon } from "lucide-react";

interface ImagePreviewProps {
  domainId: number;
  path: string;
  name: string;
}

export function ImagePreview({ domainId, path, name }: ImagePreviewProps) {
  const url = getDownloadUrl(domainId, path);
  const token = getAccessToken();

  return (
    <div className="flex flex-col items-center justify-center h-full gap-4 p-8">
      <div className="flex items-center gap-2 text-muted-foreground">
        <ImageIcon className="h-4 w-4" />
        <span className="text-sm font-mono">{name}</span>
      </div>
      <div className="max-w-full max-h-[70vh] overflow-auto rounded-lg border bg-muted/20 p-2">
        <img
          src={`${url}&token=${encodeURIComponent(token ?? "")}`}
          alt={name}
          className="max-w-full h-auto object-contain"
          onError={(e) => {
            (e.target as HTMLImageElement).style.display = "none";
          }}
        />
      </div>
    </div>
  );
}
