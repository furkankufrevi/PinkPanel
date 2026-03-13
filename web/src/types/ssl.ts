export interface SecuredComponent {
  name: string;
  secured: boolean;
}

export interface SecuredComponents {
  domain: SecuredComponent;
  www: SecuredComponent;
  mail: SecuredComponent;
  webmail: SecuredComponent;
  wildcard: SecuredComponent;
  mail_services: SecuredComponent;
}

export interface SSLCertificate {
  installed: boolean;
  id?: number;
  type?: "letsencrypt" | "custom";
  issuer?: string | null;
  domains?: string | null;
  issued_at?: string | null;
  expires_at?: string;
  auto_renew?: boolean;
  force_https?: boolean;
  hsts?: boolean;
  mail_ssl?: boolean;
  challenge_type?: "http-01" | "dns-01";
  created_at?: string;
  secured_components?: SecuredComponents;
}

export interface InstallSSLRequest {
  certificate: string;
  private_key: string;
  chain?: string;
  force_https?: boolean;
}

export interface IssueLetsEncryptRequest {
  secure_domain: boolean;
  secure_wildcard: boolean;
  include_www: boolean;
  secure_webmail: boolean;
  secure_mail: boolean;
  assign_to_mail: boolean;
}
