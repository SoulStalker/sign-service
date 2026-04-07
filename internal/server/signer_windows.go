//go:build windows

package server

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

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

func (w *WindowsSigner) ListCerts() ([]CertInfo, error) {
	certs, err := sign.ListCerts()
	if err != nil {
		return nil, err
	}
	result := make([]CertInfo, 0, len(certs))
	for _, c := range certs {
		fp := sha1.Sum(c.Raw)
		result = append(result, CertInfo{
			Thumbprint: strings.ToUpper(hex.EncodeToString(fp[:])),
			Subject:    c.Subject.String(),
			NotBefore:  c.NotBefore.UTC().Format(time.RFC3339),
			NotAfter:   c.NotAfter.UTC().Format(time.RFC3339),
		})
	}
	return result, nil
}
