import { useOutletContext } from "react-router-dom";
import { Globe, FolderOpen, Code, Shield } from "lucide-react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { Domain } from "@/types/domain";

interface DomainContext {
  domain: Domain | undefined;
  isLoading: boolean;
}

export function DomainOverview() {
  const { domain, isLoading } = useOutletContext<DomainContext>();

  const cards = [
    {
      title: "Domain",
      icon: Globe,
      value: domain?.name,
    },
    {
      title: "Document Root",
      icon: FolderOpen,
      value: domain?.document_root,
    },
    {
      title: "PHP Version",
      icon: Code,
      value: domain ? `PHP ${domain.php_version}` : undefined,
    },
    {
      title: "SSL",
      icon: Shield,
      value: "Not configured",
    },
  ];

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {cards.map((card) => (
        <Card key={card.title}>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {card.title}
            </CardTitle>
            <card.icon className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <Skeleton className="h-5 w-32" />
            ) : (
              <p className="text-sm font-medium truncate">{card.value}</p>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
