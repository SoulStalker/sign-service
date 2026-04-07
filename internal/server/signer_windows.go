//go:build windows

package server

import (
	"encoding/base64"

	"github.com/SoulStalker/sign-service/internal/sign"
)

// WindowsSigner — реализация Signer через internal/sign (crypt32.dll).
type WindowsSigner struct{}

func NewWindowsSigner() Signer {
	return &WindowsSigner{}
}

func (w *WindowsSigner) Sign(payload []byte, thumbprint string) ([]byte, error) {
	b64, err := sign.SignJSON(payload, thumbprint)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(b64)
}
