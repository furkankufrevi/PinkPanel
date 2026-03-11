import { type ColumnDef } from "@tanstack/react-table";
import { Link } from "react-router-dom";
import { MoreHorizontal, Play, Pause, Trash2 } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { StatusBadge } from "@/components/shared/status-badge";
import type { Domain } from "@/types/domain";

interface ColumnActions {
  onSuspend: (domain: Domain) => void;
  onActivate: (domain: Domain) => void;
  onDelete: (domain: Domain) => void;
}

export function getDomainColumns(actions: ColumnActions): ColumnDef<Domain, unknown>[] {
  return [
    {
      accessorKey: "name",
      header: "Domain",
      cell: ({ row }) => (
        <Link
          to={`/domains/${row.original.id}`}
          className="font-medium text-foreground hover:text-pink-500 transition-colors"
        >
          {row.original.name}
        </Link>
      ),
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
    },
    {
      accessorKey: "php_version",
      header: "PHP",
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">PHP {row.original.php_version}</span>
      ),
    },
    {
      accessorKey: "document_root",
      header: "Document Root",
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground font-mono truncate max-w-[200px] block">
          {row.original.document_root}
        </span>
      ),
    },
    {
      accessorKey: "created_at",
      header: "Created",
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {new Date(row.original.created_at).toLocaleDateString()}
        </span>
      ),
    },
    {
      id: "actions",
      header: "",
      enableSorting: false,
      cell: ({ row }) => {
        const domain = row.original;
        return (
          <DropdownMenu>
            <DropdownMenuTrigger className="flex h-8 w-8 items-center justify-center rounded-md hover:bg-accent outline-none cursor-pointer">
              <MoreHorizontal className="h-4 w-4" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {domain.status === "active" ? (
                <DropdownMenuItem onClick={() => actions.onSuspend(domain)}>
                  <Pause className="mr-2 h-4 w-4" />
                  Suspend
                </DropdownMenuItem>
              ) : (
                <DropdownMenuItem onClick={() => actions.onActivate(domain)}>
                  <Play className="mr-2 h-4 w-4" />
                  Activate
                </DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => actions.onDelete(domain)}
                className="text-destructive"
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
    },
  ];
}
