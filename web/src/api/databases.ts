import api from "@/api/client";
import type {
  Database,
  DatabaseUser,
  CreateDatabaseRequest,
  CreateDatabaseUserRequest,
} from "@/types/database";

export async function listDatabases(domainId?: number): Promise<{ data: Database[] }> {
  const params = domainId ? { domain_id: domainId } : {};
  const { data } = await api.get("/databases", { params });
  return data;
}

export async function getDatabase(id: number): Promise<{ database: Database; users: DatabaseUser[] }> {
  const { data } = await api.get(`/databases/${id}`);
  return data;
}

export async function createDatabase(req: CreateDatabaseRequest): Promise<Database> {
  const { data } = await api.post("/databases", req);
  return data;
}

export async function deleteDatabase(id: number): Promise<void> {
  await api.delete(`/databases/${id}`);
}

export async function createDatabaseUser(
  databaseId: number,
  req: CreateDatabaseUserRequest
): Promise<DatabaseUser> {
  const { data } = await api.post(`/databases/${databaseId}/users`, req);
  return data;
}

export async function deleteDatabaseUser(
  databaseId: number,
  userId: number
): Promise<void> {
  await api.delete(`/databases/${databaseId}/users/${userId}`);
}
