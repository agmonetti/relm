package conn

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SavedConnection es una conexión persistida para reutilizar.
type SavedConnection struct {
	Name     string `json:"name"`
	Driver   Driver `json:"driver"`
	Path     string `json:"path,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Database string `json:"database,omitempty"`
}

// ToConfig convierte una conexión guardada a ConnectionConfig.
func (s SavedConnection) ToConfig() ConnectionConfig {
	cfg := ConnectionConfig{
		Driver:   s.Driver,
		Name:     s.Name,
		Path:     s.Path,
		Host:     s.Host,
		Port:     s.Port,
		User:     s.User,
		Password: s.Password,
		Database: s.Database,
	}
	if cfg.Port == 0 {
		cfg.Port = DefaultPort(cfg.Driver)
	}
	return cfg
}

// FromConfig crea una SavedConnection a partir de una config.
func FromConfig(cfg ConnectionConfig) SavedConnection {
	return SavedConnection{
		Name:     cfg.Name,
		Driver:   cfg.Driver,
		Path:     cfg.Path,
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		Database: cfg.Database,
	}
}

// savedPath devuelve la ruta del archivo de conexiones guardadas.
func savedPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "relm", "connections.json"), nil
}

// LoadSaved lee las conexiones guardadas. Devuelve lista vacía si no existe.
func LoadSaved() ([]SavedConnection, error) {
	path, err := savedPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var conns []SavedConnection
	if err := json.Unmarshal(data, &conns); err != nil {
		return nil, err
	}
	return conns, nil
}

// SaveSaved escribe las conexiones guardadas con permisos 0600.
func SaveSaved(conns []SavedConnection) error {
	path, err := savedPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(conns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// SaveNamed agrega o actualiza una conexión en la lista por nombre.
func SaveNamed(conns []SavedConnection, cfg ConnectionConfig) []SavedConnection {
	sc := FromConfig(cfg)
	for i := range conns {
		if conns[i].Name == sc.Name {
			conns[i] = sc
			return conns
		}
	}
	return append(conns, sc)
}
