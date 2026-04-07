package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/SoulStalker/sign-service/gen/signer"
	"github.com/SoulStalker/sign-service/internal/config"
	"github.com/SoulStalker/sign-service/internal/server"
)

func main() {
	cfgPath := flag.String("config", "config/prod.yml", "путь к YAML-конфигу")
	flag.Parse()

	// --- Логгер ---
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// --- Конфиг ---
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Error("не удалось загрузить конфиг", "err", err)
		os.Exit(1)
	}

	// Переключаем уровень логирования согласно конфигу
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		log.Warn("неизвестный log_level, используется info", "value", cfg.LogLevel)
	}
	log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	// --- mTLS ---
	creds, err := buildTLS(cfg)
	if err != nil {
		log.Error("ошибка настройки mTLS", "err", err)
		os.Exit(1)
	}

	// --- gRPC-сервер ---
	grpcSrv := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterSignerServer(grpcSrv, server.New(newSigner(), cfg.AuditLog, log))

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Error("не удалось занять порт", "addr", cfg.GRPCAddr, "err", err)
		os.Exit(1)
	}

	// --- Graceful shutdown ---
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("сервис запущен", "addr", cfg.GRPCAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Error("gRPC Serve завершился с ошибкой", "err", err)
		}
	}()

	<-ctx.Done()
	log.Info("получен сигнал завершения, останавливаем сервис...")
	grpcSrv.GracefulStop()
	log.Info("сервис остановлен")
}

// buildTLS собирает mTLS-конфигурацию: серверный сертификат + проверка клиентского CA.
func buildTLS(cfg *config.Config) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return nil, err
	}

	caPEM, err := os.ReadFile(cfg.TLSCAFile)
	if err != nil {
		return nil, err
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, err
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS12,
	}
	return credentials.NewTLS(tlsCfg), nil
}
