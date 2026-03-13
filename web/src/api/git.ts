import api from "./client";
import type {
  GitRepository,
  GitDeployment,
  CreateGitRepoRequest,
  UpdateGitRepoRequest,
} from "@/types/git";

export async function listGitRepos(domainId: number) {
  const { data } = await api.get<{ data: GitRepository[] }>(
    `/domains/${domainId}/git`
  );
  return data;
}

export async function getGitRepo(domainId: number, repoId: number) {
  const { data } = await api.get<GitRepository>(
    `/domains/${domainId}/git/${repoId}`
  );
  return data;
}

export async function createGitRepo(
  domainId: number,
  req: CreateGitRepoRequest
) {
  const { data } = await api.post<GitRepository>(
    `/domains/${domainId}/git`,
    req
  );
  return data;
}

export async function updateGitRepo(
  domainId: number,
  repoId: number,
  req: UpdateGitRepoRequest
) {
  const { data } = await api.put<GitRepository>(
    `/domains/${domainId}/git/${repoId}`,
    req
  );
  return data;
}

export async function deleteGitRepo(domainId: number, repoId: number) {
  const { data } = await api.delete(`/domains/${domainId}/git/${repoId}`);
  return data;
}

export async function triggerDeploy(domainId: number, repoId: number) {
  const { data } = await api.post<GitDeployment>(
    `/domains/${domainId}/git/${repoId}/deploy`
  );
  return data;
}

export async function listDeployments(
  domainId: number,
  repoId: number,
  limit?: number
) {
  const params = limit ? { limit } : {};
  const { data } = await api.get<{ data: GitDeployment[] }>(
    `/domains/${domainId}/git/${repoId}/deployments`,
    { params }
  );
  return data;
}
