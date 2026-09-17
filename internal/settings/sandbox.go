package settings

// SandboxTarget holds the configuration options for a single sandbox target.
type SandboxTarget struct {
	Backend      string `json:"backend" yaml:"backend"`
	Host         string `json:"host" yaml:"host"`
	VMName       string `json:"vm_name" yaml:"vm_name"`
	Port         int    `json:"port" yaml:"port"`
	User         string `json:"user" yaml:"user"`
	SSHKeyPath   string `json:"ssh_key_path" yaml:"ssh_key_path"`
	Root         string `json:"root" yaml:"root"`
	Workdir      string `json:"workdir" yaml:"workdir"`
	AutoSync     *bool  `json:"auto_sync" yaml:"auto_sync"`
	AutoSnapshot *bool  `json:"auto_snapshot" yaml:"auto_snapshot"`
}

// Sandbox holds the configuration options for sandbox environments.
type Sandbox struct {
	Default      string                   `json:"default" yaml:"default"`
	User         string                   `json:"user" yaml:"user"`
	Targets      map[string]SandboxTarget `json:"targets" yaml:"targets"`
	VMName       string                   `json:"vm_name" yaml:"vm_name"`
	Port         int                      `json:"port" yaml:"port"`
	SSHKeyPath   string                   `json:"ssh_key_path" yaml:"ssh_key_path"`
	Root         string                   `json:"root" yaml:"root"`
	Workdir      string                   `json:"workdir" yaml:"workdir"`
	AutoSync     *bool                    `json:"auto_sync" yaml:"auto_sync"`
	AutoSnapshot *bool                    `json:"auto_snapshot" yaml:"auto_snapshot"`
}

func (t *SandboxTarget) merge(over SandboxTarget) {
	if over.Backend != "" {
		t.Backend = over.Backend
	}
	if over.Host != "" {
		t.Host = over.Host
	}
	if over.VMName != "" {
		t.VMName = over.VMName
	}
	if over.Port != 0 {
		t.Port = over.Port
	}
	if over.User != "" {
		t.User = over.User
	}
	if over.SSHKeyPath != "" {
		t.SSHKeyPath = over.SSHKeyPath
	}
	if over.Root != "" {
		t.Root = over.Root
	}
	if over.Workdir != "" {
		t.Workdir = over.Workdir
	}
	mergeBool(&t.AutoSync, over.AutoSync)
	mergeBool(&t.AutoSnapshot, over.AutoSnapshot)
}

func (s *Sandbox) mergeBasic(over Sandbox) {
	if over.Default != "" {
		s.Default = over.Default
	}
	if over.User != "" {
		s.User = over.User
	}
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

func (s *Sandbox) mergeTargets(over map[string]SandboxTarget) {
	if len(over) == 0 {
		return
	}
	if s.Targets == nil {
		s.Targets = make(map[string]SandboxTarget, len(over))
	}
	for k, v := range over {
		existing, ok := s.Targets[k]
		if !ok {
			s.Targets[k] = v
			continue
		}
		existing.merge(v)
		s.Targets[k] = existing
	}
}

func (s *Sandbox) merge(over Sandbox) {
	s.mergeBasic(over)
	s.mergeTargets(over.Targets)
}
