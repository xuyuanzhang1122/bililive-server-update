package model

import "time"

type Catalog struct {
	UpdatedAt time.Time         `json:"updated_at"`
	Releases  []ReleaseArtifact `json:"releases"`
	Tools     []ToolArtifact    `json:"tools"`
}

type ReleaseArtifact struct {
	Source      string            `json:"source"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Channel     string            `json:"channel"`
	InstallMode string            `json:"install_mode"`
	OS          string            `json:"os,omitempty"`
	Arch        string            `json:"arch,omitempty"`
	URL         string            `json:"url"`
	SHA256      string            `json:"sha256,omitempty"`
	Size        int64             `json:"size,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type ToolArtifact struct {
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	OS        string            `json:"os,omitempty"`
	Arch      string            `json:"arch,omitempty"`
	URL       string            `json:"url"`
	SHA256    string            `json:"sha256,omitempty"`
	Stable    bool              `json:"stable"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type InstallManifest struct {
	PublicBaseURL string          `json:"public_base_url,omitempty"`
	GeneratedAt   time.Time       `json:"generated_at"`
	Sources       []InstallSource `json:"sources"`
	Modes         []InstallMode   `json:"modes"`
	RequiredTools []RequiredTool  `json:"required_tools"`
	DoctorAPI     string          `json:"doctor_api"`
	BackupAPI     string          `json:"backup_api"`
	Catalog       Catalog         `json:"catalog"`
}

type InstallSource struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Default     bool   `json:"default,omitempty"`
}

type InstallMode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Deprecated  bool   `json:"deprecated,omitempty"`
}

type RequiredTool struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Required    bool     `json:"required"`
	ConfigKey   string   `json:"config_key,omitempty"`
	EnvKeys     []string `json:"env_keys,omitempty"`
	Description string   `json:"description"`
}

type DoctorRequest struct {
	OS          string            `json:"os"`
	Arch        string            `json:"arch"`
	InstallMode string            `json:"install_mode"`
	Port        int               `json:"port"`
	Paths       map[string]string `json:"paths"`
	Tools       []ClientTool      `json:"tools"`
}

type ClientTool struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
	OK      bool   `json:"ok"`
}

type DoctorResponse struct {
	OK     bool          `json:"ok"`
	Checks []DoctorCheck `json:"checks"`
}

type DoctorCheck struct {
	ID      string `json:"id"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type BackupBundle struct {
	ID            string             `json:"id"`
	DeviceName    string             `json:"device_name,omitempty"`
	AppVersion    string             `json:"app_version,omitempty"`
	Server        BackupServerConfig `json:"server"`
	Rooms         []BackupLiveRoom   `json:"rooms"`
	PackageBase64 string             `json:"package_base64,omitempty"`
	ClientHash    string             `json:"client_hash,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	SchemaVersion int                `json:"schema_version"`
}

type BackupServerConfig struct {
	BaseURL     string `json:"base_url,omitempty"`
	RPCBind     string `json:"rpc_bind,omitempty"`
	Port        int    `json:"port,omitempty"`
	OutputPath  string `json:"output_path,omitempty"`
	AppDataPath string `json:"app_data_path,omitempty"`
	ConfigPath  string `json:"config_path,omitempty"`
}

type BackupLiveRoom struct {
	URL         string `json:"url"`
	IsListening bool   `json:"is_listening"`
	Scheme      string `json:"scheme,omitempty"`
}

type RestoreRequest struct {
	TargetServerURL string `json:"target_server_url,omitempty"`
	DryRun          bool   `json:"dry_run"`
}

type RestorePlan struct {
	BackupID        string   `json:"backup_id"`
	DryRun          bool     `json:"dry_run"`
	Steps           []string `json:"steps"`
	RequiresLocal   bool     `json:"requires_local"`
	LocalToolAPI    string   `json:"local_tool_api"`
	TargetServerURL string   `json:"target_server_url,omitempty"`
}
