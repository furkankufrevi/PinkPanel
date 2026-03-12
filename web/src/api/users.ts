import api from "./client";

export interface User {
  id: number;
  username: string;
  email: string;
  role: "super_admin" | "admin" | "user";
  status: "active" | "suspended";
  system_username?: string;
  created_at: string;
  updated_at: string;
}

export interface UserWithStats extends User {
  domain_count: number;
  database_count: number;
  ftp_count: number;
}

export interface CreateUserRequest {
  username: string;
  email: string;
  password: string;
  role: string;
}

export interface UpdateUserRequest {
  email?: string;
  role?: string;
}

export async function listUsers(search?: string): Promise<UserWithStats[]> {
  const params = search ? { search } : {};
  const response = await api.get<{ data: UserWithStats[] }>("/users", { params });
  return response.data.data;
}

export async function getUser(id: number): Promise<User> {
  const response = await api.get<User>(`/users/${id}`);
  return response.data;
}

export async function createUser(data: CreateUserRequest): Promise<User> {
  const response = await api.post<User>("/users", data);
  return response.data;
}

export async function updateUser(id: number, data: UpdateUserRequest): Promise<User> {
  const response = await api.put<User>(`/users/${id}`, data);
  return response.data;
}

export async function deleteUser(id: number): Promise<void> {
  await api.delete(`/users/${id}`);
}

export async function suspendUser(id: number): Promise<User> {
  const response = await api.post<User>(`/users/${id}/suspend`);
  return response.data;
}

export async function activateUser(id: number): Promise<User> {
  const response = await api.post<User>(`/users/${id}/activate`);
  return response.data;
}

export async function resetUserPassword(id: number, password: string): Promise<void> {
  await api.post(`/users/${id}/reset-password`, { password });
}

export async function getProfile(): Promise<User> {
  const response = await api.get<User>("/auth/profile");
  return response.data;
}
