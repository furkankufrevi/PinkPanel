export interface Database {
  id: number;
  domain_id: number | null;
  name: string;
  type: string;
  size_bytes: number;
  created_at: string;
  updated_at: string;
}

export interface DatabaseUser {
  id: number;
  database_id: number;
  username: string;
  host: string;
  permissions: string;
  created_at: string;
}

export interface CreateDatabaseRequest {
  name: string;
  domain_id?: number | null;
}

export interface CreateDatabaseUserRequest {
  username: string;
  password: string;
  host?: string;
  permissions?: string;
}
