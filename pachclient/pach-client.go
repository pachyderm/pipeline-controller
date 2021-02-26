package pachclient

import (
	"github.com/pachyderm/pachyderm/v2/src/client"
)

func Connect() (*client.APIClient, error) {
	return client.NewFromAddress("localhost:30650")
}
