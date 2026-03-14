package app

// AppDefinition describes an installable application.
type AppDefinition struct {
	Slug         string   `json:"slug"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Category     string   `json:"category"`
	Icon         string   `json:"icon"`
	Website      string   `json:"website"`
	DownloadURL  string   `json:"download_url"`
	ArchiveFormat string  `json:"archive_format"`
	ExtractSubdir string  `json:"extract_subdir"`
	MinPHP       string   `json:"min_php"`
	RequiredExts []string `json:"required_exts"`
	NeedsDB      bool     `json:"needs_db"`
	HasCLI       bool     `json:"has_cli"`
	ConfigFile   string   `json:"config_file"`
	VersionCmd   string   `json:"version_cmd"`
}

var catalog = []AppDefinition{
	{
		Slug:          "wordpress",
		Name:          "WordPress",
		Description:   "The world's most popular content management system",
		Category:      "cms",
		Icon:          "wordpress",
		Website:       "https://wordpress.org",
		DownloadURL:   "https://wordpress.org/latest.tar.gz",
		ArchiveFormat: "tar.gz",
		ExtractSubdir: "wordpress",
		MinPHP:        "7.4",
		RequiredExts:  []string{"mysqli", "curl", "gd", "mbstring", "xml", "zip"},
		NeedsDB:       true,
		HasCLI:        true,
		ConfigFile:    "wp-config.php",
		VersionCmd:    "core version",
	},
	{
		Slug:          "joomla",
		Name:          "Joomla",
		Description:   "Flexible and powerful CMS for building websites",
		Category:      "cms",
		Icon:          "joomla",
		Website:       "https://www.joomla.org",
		DownloadURL:   "https://downloads.joomla.org/cms/joomla5/5-2-4/Joomla_5-2-4-Stable-Full_Package.zip",
		ArchiveFormat: "zip",
		ExtractSubdir: "",
		MinPHP:        "8.1",
		RequiredExts:  []string{"json", "simplexml", "dom", "zlib", "gd", "mysqli"},
		NeedsDB:       true,
		HasCLI:        false,
		ConfigFile:    "configuration.php",
	},
	{
		Slug:          "drupal",
		Name:          "Drupal",
		Description:   "Enterprise-grade CMS for ambitious digital experiences",
		Category:      "cms",
		Icon:          "drupal",
		Website:       "https://www.drupal.org",
		DownloadURL:   "https://www.drupal.org/download-latest/tar.gz",
		ArchiveFormat: "tar.gz",
		ExtractSubdir: "auto",
		MinPHP:        "8.1",
		RequiredExts:  []string{"pdo_mysql", "gd", "xml", "mbstring", "curl", "json"},
		NeedsDB:       true,
		HasCLI:        false,
		ConfigFile:    "sites/default/settings.php",
	},
	{
		Slug:          "prestashop",
		Name:          "PrestaShop",
		Description:   "Open-source e-commerce platform for online stores",
		Category:      "ecommerce",
		Icon:          "prestashop",
		Website:       "https://www.prestashop-project.org",
		DownloadURL:   "https://github.com/PrestaShop/PrestaShop/releases/latest/download/prestashop_8.2.0.zip",
		ArchiveFormat: "zip",
		ExtractSubdir: "",
		MinPHP:        "8.1",
		RequiredExts:  []string{"curl", "gd", "intl", "mbstring", "xml", "zip", "pdo_mysql"},
		NeedsDB:       true,
		HasCLI:        false,
		ConfigFile:    "config/settings.inc.php",
	},
	{
		Slug:          "phpmyadmin",
		Name:          "phpMyAdmin",
		Description:   "Web-based MySQL/MariaDB database administration tool",
		Category:      "tools",
		Icon:          "phpmyadmin",
		Website:       "https://www.phpmyadmin.net",
		DownloadURL:   "https://www.phpmyadmin.net/downloads/phpMyAdmin-latest-all-languages.tar.gz",
		ArchiveFormat: "tar.gz",
		ExtractSubdir: "auto",
		MinPHP:        "8.1",
		RequiredExts:  []string{"mysqli", "mbstring", "json", "xml"},
		NeedsDB:       false,
		HasCLI:        false,
		ConfigFile:    "config.inc.php",
	},
}

// GetCatalog returns all available app definitions.
func GetCatalog() []AppDefinition {
	return catalog
}

// GetAppDef looks up an app definition by slug.
func GetAppDef(slug string) *AppDefinition {
	for i := range catalog {
		if catalog[i].Slug == slug {
			return &catalog[i]
		}
	}
	return nil
}
