package scimserverlite

// DummyBackend is a simple SCIM backend which doesn't remember anything
type DummyBackend struct {
	parser ObjectParser
}

// NewDummyBackend allocates and returns a new DummyBackend
func NewDummyBackend(parser ObjectParser) *DummyBackend {
	var backend DummyBackend
	backend.parser = parser
	return &backend
}

func (backend *DummyBackend) Create(tenant, resourceType, resource string) (string, error) {
	_, err := backend.parser(resourceType, resource)
	if err != nil {
		return "", NewError(MalformedResourceError, "Failed to parse resource:\n"+err.Error())
	}
	return resource, nil
}

func (backend *DummyBackend) Update(tenant, resourceType, resourceID, resource string) (string, error) {
	_, err := backend.parser(resourceType, resource)
	if err != nil {
		return "", NewError(MalformedResourceError, "Failed to parse resource:\n"+err.Error())
	}
	return resource, nil
}

func (backend *DummyBackend) Delete(tenant, resourceType, resourceID string) error {
	return nil
}

func (backend *DummyBackend) Clear(tenant string) error {
	return nil
}

func (backend *DummyBackend) GetResources(tenant, resourceType string) (map[string]string, error) {
	return make(map[string]string), nil
}

func (backend *DummyBackend) GetResource(tenant, resourceType string, id string) (string, error) {
	return "", NewError(MissingResourceError, "Resource missing: "+id)
}

func (backend *DummyBackend) GetParsedResources(tenant, resourceType string) (map[string]any, error) {
	return make(map[string]any), nil
}

func (backend *DummyBackend) GetParsedResource(tenant, resourceType string, id string) (interface{}, error) {
	return nil, NewError(MissingResourceError, "Resource missing: "+id)
}
