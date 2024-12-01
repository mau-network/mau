package mau

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUPNPClient(t *testing.T) {
	t.Skip("This test proven to be slow as it asks the network gateway for a port")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client, err := newUPNPClient(ctx)
	assert.NoError(t, err)

	address, err := client.GetExternalIPAddress()
	assert.NoError(t, err)
	assert.NotEqual(t, "", address)
}
