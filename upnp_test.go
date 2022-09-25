package mau

import (
	"context"
	"testing"
	"time"
)

func TestUPNPClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client, err := NewUPNPClient(ctx)
	ASSERT_NO_ERROR(t, err)

	address, err := client.GetExternalIPAddress()
	ASSERT_NO_ERROR(t, err)
	REFUTE_EQUAL(t, "", address)
}
