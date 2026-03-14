export interface DNSTemplateRecord {
  id: number;
  template_id: number;
  type: string;
  name: string;
  value: string;
  ttl: number;
  priority: number | null;
}

export interface DNSTemplate {
  id: number;
  name: string;
  description: string;
  category: string;
  is_preset: boolean;
  records: DNSTemplateRecord[];
  created_at?: string;
  updated_at?: string;
}

export interface CreateDNSTemplateRequest {
  name: string;
  description: string;
  category: string;
  records: Omit<DNSTemplateRecord, "id" | "template_id">[];
}

export interface ApplyTemplateRequest {
  template_id: number;
  mode: "merge" | "replace";
}

export interface SaveAsTemplateRequest {
  name: string;
  description: string;
}
