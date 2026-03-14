export interface AppDefinition {
  slug: string;
  name: string;
  description: string;
  category: string;
  icon: string;
  website: string;
  download_url: string;
  archive_format: string;
  extract_subdir: string;
  min_php: string;
  required_exts: string[];
  needs_db: boolean;
  has_cli: boolean;
  config_file: string;
  version_cmd: string;
}

export interface InstalledApp {
  id: number;
  domain_id: number;
  app_type: string;
  app_name: string;
  version: string;
  install_path: string;
  db_name: string | null;
  db_user: string | null;
  admin_url: string | null;
  status:
    | "pending"
    | "installing"
    | "completed"
    | "failed"
    | "updating"
    | "uninstalling";
  error_message: string | null;
  install_log?: string | null;
  installed_at: string;
  updated_at: string;
}

export interface InstallAppRequest {
  app_type: string;
  site_title?: string;
  admin_user?: string;
  admin_pass?: string;
  admin_email?: string;
  db_name?: string;
  db_user?: string;
  db_pass?: string;
  install_path?: string;
}

export interface WPInfo {
  version: string;
  plugins_json: string;
  themes_json: string;
}

export interface WPPlugin {
  name: string;
  status: string;
  update: string;
  version: string;
}

export interface WPTheme {
  name: string;
  status: string;
  update: string;
  version: string;
}
