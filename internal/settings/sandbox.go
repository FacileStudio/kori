package settings

// Sandbox holds the configuration options for boite VM sandbox environments.
type Sandbox struct {
	VMName       string `json:"vm_name" yaml:"vm_name"`
	Port         int    `json:"port" yaml:"port"`
	SSHKeyPath   string `json:"ssh_key_path" yaml:"ssh_key_path"`
	Root         string `json:"root" yaml:"root"`
	Workdir      string `json:"workdir" yaml:"workdir"`
	AutoSync     *bool  `json:"auto_sync" yaml:"auto_sync"`
	AutoSnapshot *bool  `json:"auto_snapshot" yaml:"auto_snapshot"`
}

func (s *Sandbox) merge(over Sandbox) {
	if over.VMName != "" {
		s.VMName = over.VMName
	}
	if over.Port != 0 {
		s.Port = over.Port
	}
	if over.SSHKeyPath != "" {
		s.SSHKeyPath = over.SSHKeyPath
	}
	if over.Root != "" {
		s.Root = over.Root
	}
	if over.Workdir != "" {
		s.Workdir = over.Workdir
	}
	mergeBool(&s.AutoSync, over.AutoSync)
	mergeBool(&s.AutoSnapshot, over.AutoSnapshot)
}
