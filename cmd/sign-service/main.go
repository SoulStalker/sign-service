package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

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

	// --- gRPC-сервер ---
	grpcSrv := grpc.NewServer()
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
