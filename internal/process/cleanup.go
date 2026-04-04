package process

import "os"

func (m *Manager) Cleanup() error {
	if m.tmpDir == "" {
		return nil
	}

	err := os.RemoveAll(m.tmpDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
