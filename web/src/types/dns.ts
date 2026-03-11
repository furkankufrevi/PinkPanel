export interface DNSRecord {
  id: number;
  domain_id: number;
  type: string;
  name: string;
  value: string;
  ttl: number;
  priority: number | null;
  created_at: string;
  updated_at: string;
}

export interface CreateDNSRecordRequest {
  type: string;
  name: string;
  value: string;
  ttl: number;
  priority?: number | null;
}

export interface UpdateDNSRecordRequest {
  type: string;
  name: string;
  value: string;
  ttl: number;
  priority?: number | null;
}
