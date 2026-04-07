//go:build !windows

package main

import (
	"errors"

	"github.com/SoulStalker/sign-service/internal/server"
)

// stubSigner — заглушка для не-Windows сборок (сервис работает только на Windows).
type stubSigner struct{}

func (stubSigner) Sign(_ []byte, _ string) ([]byte, error) {
	return nil, errors.New("подпись поддерживается только на Windows")
}

func (stubSigner) ListCerts() ([]server.CertInfo, error) {
	return nil, errors.New("ListCerts поддерживается только на Windows")
}

func newSigner() server.Signer {
	return stubSigner{}
}
