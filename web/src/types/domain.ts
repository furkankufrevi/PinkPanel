export interface Domain {
  id: number;
  name: string;
  document_root: string;
  status: "active" | "suspended";
  php_version: string;
  parent_id: number | null;
  separate_dns: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateDomainRequest {
  name: string;
  php_version: string;
  create_www: boolean;
  parent_id?: number;
}

export interface UpdateDomainRequest {
  document_root?: string;
  php_version?: string;
  separate_dns?: boolean;
}

export interface DomainListResponse {
  data: Domain[];
  total: number;
  page: number;
  per_page: number;
}
