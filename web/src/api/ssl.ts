import api from "@/api/client";
import type { SSLCertificate, InstallSSLRequest } from "@/types/ssl";

export async function getSSLCertificate(domainId: number): Promise<SSLCertificate> {
  const { data } = await api.get(`/domains/${domainId}/ssl`);
  return data;
}

export async function installSSLCertificate(
  domainId: number,
  req: InstallSSLRequest
): Promise<SSLCertificate> {
  const { data } = await api.post(`/domains/${domainId}/ssl`, req);
  return data;
}

export async function deleteSSLCertificate(domainId: number): Promise<void> {
  await api.delete(`/domains/${domainId}/ssl`);
}

export async function toggleSSLAutoRenew(
  domainId: number,
  enabled: boolean
): Promise<void> {
  await api.put(`/domains/${domainId}/ssl/auto-renew`, { enabled });
}

export async function issueLetsEncrypt(
  domainId: number,
  includeWww: boolean = true
): Promise<SSLCertificate> {
  const { data } = await api.post(`/domains/${domainId}/ssl/issue`, {
    include_www: includeWww,
  });
  return data;
}
