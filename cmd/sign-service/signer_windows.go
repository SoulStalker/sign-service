//go:build windows

package main

import "github.com/SoulStalker/sign-service/internal/server"

func newSigner() server.Signer {
	return server.NewWindowsSigner()
}
