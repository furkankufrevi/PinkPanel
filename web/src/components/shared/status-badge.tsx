import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

const variants: Record<string, { className: string; label: string }> = {
  active: { className: "bg-emerald-500/10 text-emerald-500 border-emerald-500/20", label: "Active" },
  suspended: { className: "bg-amber-500/10 text-amber-500 border-amber-500/20", label: "Suspended" },
  error: { className: "bg-red-500/10 text-red-500 border-red-500/20", label: "Error" },
  pending: { className: "bg-blue-500/10 text-blue-500 border-blue-500/20", label: "Pending" },
  running: { className: "bg-blue-500/10 text-blue-500 border-blue-500/20", label: "Running" },
  completed: { className: "bg-emerald-500/10 text-emerald-500 border-emerald-500/20", label: "Completed" },
  failed: { className: "bg-red-500/10 text-red-500 border-red-500/20", label: "Failed" },
};

interface StatusBadgeProps {
  status: string;
  label?: string;
  className?: string;
}

export function StatusBadge({ status, label, className }: StatusBadgeProps) {
  const variant = variants[status] ?? { className: "bg-muted text-muted-foreground", label: status };
  return (
    <Badge variant="outline" className={cn("font-medium", variant.className, className)}>
      {label ?? variant.label}
    </Badge>
  );
}
