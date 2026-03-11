export interface Domain {
  id: number;
  name: string;
  document_root: string;
  status: "active" | "suspended";
  php_version: string;
  created_at: string;
  updated_at: string;
}

export interface CreateDomainRequest {
  name: string;
  php_version: string;
  create_www: boolean;
}

export interface UpdateDomainRequest {
  document_root?: string;
  php_version?: string;
}

export interface DomainListResponse {
  data: Domain[];
  total: number;
  page: number;
  per_page: number;
}
