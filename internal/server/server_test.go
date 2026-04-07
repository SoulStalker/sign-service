//go:build !windows

package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/SoulStalker/sign-service/gen/signer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// stubSigner — заглушка Signer для тестов.
type stubSigner struct {
	sig []byte
	err error
}

func (s *stubSigner) Sign(_ []byte, _ string) ([]byte, error) {
	return s.sig, s.err
}

func (s *stubSigner) ListCerts() ([]CertInfo, error) {
	return nil, nil
}

// helpers

func newTestServer(t *testing.T, signer Signer) (*SignerServer, string) {
	t.Helper()
	auditFile := filepath.Join(t.TempDir(), "audit.jsonl")
	srv := New(signer, auditFile, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	return srv, auditFile
}

func readAuditEntries(t *testing.T, path string) []auditEntry {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		return nil // файл ещё не создан
	}
	defer f.Close()

	var entries []auditEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e auditEntry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("не удалось разобрать строку аудит-лога: %v", err)
		}
		entries = append(entries, e)
	}
	return entries
}

func grpcCode(err error) codes.Code {
	if s, ok := status.FromError(err); ok {
		return s.Code()
	}
	return codes.Unknown
}

// тесты

func TestSign_Success(t *testing.T) {
	want := []byte("fake-signature")
	srv, auditFile := newTestServer(t, &stubSigner{sig: want})

	resp, err := srv.Sign(context.Background(), &pb.SignRequest{
		Payload:    []byte("hello"),
		Thumbprint: "aabbccdd",
		CallerId:   "edo-client",
	})
	if err != nil {
		t.Fatalf("Sign вернул ошибку: %v", err)
	}
	if string(resp.Signature) != string(want) {
		t.Errorf("подпись: got %q, want %q", resp.Signature, want)
	}

	entries := readAuditEntries(t, auditFile)
	if len(entries) != 1 {
		t.Fatalf("ожидалась 1 запись аудита, получили %d", len(entries))
	}
	e := entries[0]
	if !e.Ok {
		t.Error("аудит: ok должен быть true")
	}
	if e.Caller != "edo-client" {
		t.Errorf("аудит: caller = %q, want %q", e.Caller, "edo-client")
	}
	if e.Thumbprint != "aabbccdd" {
		t.Errorf("аудит: thumbprint = %q, want %q", e.Thumbprint, "aabbccdd")
	}
	if e.PayloadSize != len("hello") {
		t.Errorf("аудит: payload_size = %d, want %d", e.PayloadSize, len("hello"))
	}
	if e.Ts == "" {
		t.Error("аудит: ts не должен быть пустым")
	}
}

func TestSign_EmptyThumbprint(t *testing.T) {
	srv, _ := newTestServer(t, &stubSigner{sig: []byte("x")})

	_, err := srv.Sign(context.Background(), &pb.SignRequest{
		Payload:    []byte("hello"),
		Thumbprint: "",
	})
	if grpcCode(err) != codes.InvalidArgument {
		t.Errorf("ожидался InvalidArgument, получили: %v", err)
	}
}

func TestSign_EmptyPayload(t *testing.T) {
	srv, _ := newTestServer(t, &stubSigner{sig: []byte("x")})

	_, err := srv.Sign(context.Background(), &pb.SignRequest{
		Payload:    nil,
		Thumbprint: "aabbccdd",
	})
	if grpcCode(err) != codes.InvalidArgument {
		t.Errorf("ожидался InvalidArgument, получили: %v", err)
	}
}

func TestSign_SignerError(t *testing.T) {
	srv, auditFile := newTestServer(t, &stubSigner{err: errors.New("csp error")})

	_, err := srv.Sign(context.Background(), &pb.SignRequest{
		Payload:    []byte("hello"),
		Thumbprint: "aabbccdd",
		CallerId:   "chestnyznak",
	})
	if grpcCode(err) != codes.Internal {
		t.Errorf("ожидался Internal, получили: %v", err)
	}

	entries := readAuditEntries(t, auditFile)
	if len(entries) != 1 {
		t.Fatalf("ожидалась 1 запись аудита, получили %d", len(entries))
	}
	if entries[0].Ok {
		t.Error("аудит: ok должен быть false при ошибке подписи")
	}
}
