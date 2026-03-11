import { useCallback, type ReactNode } from "react";
import { useDropzone } from "react-dropzone";
import { Upload } from "lucide-react";

interface UploadZoneProps {
  children: ReactNode;
  onDrop: (files: File[]) => void;
  disabled?: boolean;
}

export function UploadZone({ children, onDrop, disabled }: UploadZoneProps) {
  const handleDrop = useCallback(
    (accepted: File[]) => {
      if (accepted.length > 0) onDrop(accepted);
    },
    [onDrop]
  );

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop: handleDrop,
    noClick: true,
    noKeyboard: true,
    disabled,
  });

  return (
    <div {...getRootProps()} className="relative h-full">
      <input {...getInputProps()} />
      {children}
      {isDragActive && (
        <div className="absolute inset-0 z-50 flex flex-col items-center justify-center gap-3 bg-background/90 backdrop-blur-sm border-2 border-dashed border-pink-500 rounded-lg">
          <Upload className="h-12 w-12 text-pink-500" />
          <p className="text-lg font-medium text-pink-500">Drop files here to upload</p>
        </div>
      )}
    </div>
  );
}
