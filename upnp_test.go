package mau

import (
	"context"
	"testing"
	"time"
)

func TestUPNPClient(t *testing.T) {
	t.Skip("This test proven to be slow as it asks the network gateway for a port")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client, err := newUPNPClient(ctx)
	ASSERT_NO_ERROR(t, err)

	address, err := client.GetExternalIPAddress()
	ASSERT_NO_ERROR(t, err)
	REFUTE_EQUAL(t, "", address)
}
