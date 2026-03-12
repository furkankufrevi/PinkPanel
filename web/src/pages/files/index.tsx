import { DomainFiles } from "@/pages/domains/detail/files";

export function FilesPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">File Manager</h1>
        <p className="text-muted-foreground">
          Browse and edit all website files in /var/www
        </p>
      </div>
      <DomainFiles domainId={0} />
    </div>
  );
}
