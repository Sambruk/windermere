package windermere

import (
	"os"

	"github.com/Sambruk/windermere/scimserverlite"
)

// Saves the in-memory backend to file
func saveSCIMBackend(backend *scimserverlite.InMemoryBackend, path string) error {
	serializedForm, err := backend.Serialize()

	if err != nil {
		return err
	}

	return os.WriteFile(path, serializedForm, 0600)
}
