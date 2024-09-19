package program

import (
	"strings"
	"testing"

	"github.com/Sambruk/windermere/test"
	"github.com/spf13/viper"
)

func TestParseClients(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")

	working := `
SkolsynkListenAddress: :8001
SkolsynkAuthHeader: X-API-Key
SkolsynkClients:
  - name: skolsynkGoogle
    key: gurka
  - name: skolsynkMicrosoft
    key: banan
`

	v.ReadConfig(strings.NewReader(working))

	clients := v.Get(CNFSkolsynkClients)
	parsed, err := parseClients(clients)
	test.Ensure(t, err)

	if len(parsed) != 2 || parsed["skolsynkGoogle"] != "gurka" || parsed["skolsynkMicrosoft"] != "banan" {
		t.Errorf("Unexpected parsed clients: %v", parsed)
	}

	broken := `
SkolsynkListenAddress: :8001
SkolsynkAuthHeader: X-API-Key
SkolsynkClients: 7
`
	v = viper.New()
	v.SetConfigType("yaml")
	v.ReadConfig(strings.NewReader(broken))

	_, err = parseClients(v.Get(CNFSkolsynkClients))
	test.MustFail(t, err)
}
