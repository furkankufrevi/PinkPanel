export interface SSLCertificate {
  installed: boolean;
  id?: number;
  type?: "letsencrypt" | "custom";
  issuer?: string | null;
  domains?: string | null;
  issued_at?: string | null;
  expires_at?: string;
  auto_renew?: boolean;
  created_at?: string;
}

export interface InstallSSLRequest {
  certificate: string;
  private_key: string;
  chain?: string;
  force_https?: boolean;
}
