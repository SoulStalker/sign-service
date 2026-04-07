package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	pb "github.com/SoulStalker/sign-service/gen/signer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Signer — интерфейс подписи. Позволяет подменять реализацию в тестах.
type Signer interface {
	Sign(payload []byte, thumbprint string) ([]byte, error)
}

// SignerServer реализует gRPC-интерфейс signer.SignerServer.
type SignerServer struct {
	pb.UnimplementedSignerServer
	signer   Signer
	auditLog string
	log      *slog.Logger
}

// New создаёт SignerServer. auditLog — путь к JSONL-файлу аудита.
func New(signer Signer, auditLog string, log *slog.Logger) *SignerServer {
	return &SignerServer{signer: signer, auditLog: auditLog, log: log}
}

// auditEntry — строка аудит-лога.
type auditEntry struct {
	Ts          string `json:"ts"`
	Caller      string `json:"caller"`
	Thumbprint  string `json:"thumbprint"`
	PayloadSize int    `json:"payload_size"`
	Ok          bool   `json:"ok"`
}

func (s *SignerServer) writeAudit(e auditEntry) {
	f, err := os.OpenFile(s.auditLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		s.log.Error("не удалось открыть аудит-лог", "err", err)
		return
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(e); err != nil {
		s.log.Error("не удалось записать аудит-лог", "err", err)
	}
}

func (s *SignerServer) Sign(ctx context.Context, req *pb.SignRequest) (*pb.SignResponse, error) {
	if req.Thumbprint == "" {
		return nil, status.Error(codes.InvalidArgument, "thumbprint обязателен")
	}
	if len(req.Payload) == 0 {
		return nil, status.Error(codes.InvalidArgument, "payload не может быть пустым")
	}

	sig, err := s.signer.Sign(req.Payload, req.Thumbprint)
	ok := err == nil

	s.writeAudit(auditEntry{
		Ts:          time.Now().UTC().Format(time.RFC3339),
		Caller:      req.CallerId,
		Thumbprint:  req.Thumbprint,
		PayloadSize: len(req.Payload),
		Ok:          ok,
	})

	if err != nil {
		s.log.Error("ошибка подписи", "caller", req.CallerId, "thumbprint", req.Thumbprint, "err", err)
		return nil, status.Errorf(codes.Internal, "ошибка подписи: %v", err)
	}

	return &pb.SignResponse{Signature: sig}, nil
}

func (s *SignerServer) Verify(_ context.Context, _ *pb.VerifyRequest) (*pb.VerifyResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Verify не реализован")
}

func (s *SignerServer) ListCertificates(_ context.Context, _ *pb.Empty) (*pb.CertListResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListCertificates не реализован")
}
