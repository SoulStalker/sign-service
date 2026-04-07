# sign-service

Windows-only gRPC-сервис криптографической подписи через Windows Certificate Store (ГОСТ-сертификаты, `crypt32.dll`).

Принимает `thumbprint` в каждом запросе — поддерживает несколько сертификатов одновременно.
Предназначен для переиспользования несколькими клиентами: `edo-client`, `chestnyznak-client` и др.

## Архитектура

```
cmd/sign-service/          Точка входа: конфиг → gRPC-сервер
internal/config/           YAML-конфиг (cleanenv)
internal/sign/             Syscall-обёртки crypt32.dll (только Windows)
internal/server/           SignerServer: реализует gRPC-интерфейс, пишет аудит-лог
gen/signer/                Сгенерированные protobuf + gRPC стабы (не редактировать)
proto/signer/signer.proto  Источник истины для gRPC-контракта
scripts/generate.ps1       Генерация gen/ из proto (Windows)
scripts/install-service.ps1  Установка как Windows-служба через NSSM
```

## Сборка и запуск

> Требуется Windows — `internal/sign` использует `crypt32.dll`.

### Зависимости для генерации stubs

Перед первым запуском `.\scripts\generate.ps1` установите `protoc`:

```powershell
# через winget
winget install Google.Protobuf

# или через Chocolatey
choco install protoc
```

Go-плагины (`protoc-gen-go`, `protoc-gen-go-grpc`) скрипт установит автоматически.

```powershell
# Генерация gRPC-стабов (после любого изменения .proto)
.\scripts\generate.ps1

# Сборка
go build -o sign-service.exe .\cmd\sign-service

# Запуск
.\sign-service.exe --config config\prod.yml
```

## Конфигурация

Конфиг читается из YAML-файла (`--config`, default: `config\prod.yml`).
Все поля можно переопределить переменными среды.

| Поле        | ENV        | Default         | Описание                   |
|-------------|------------|-----------------|----------------------------|
| `grpc_addr` | `GRPC_ADDR`| `0.0.0.0:50051` | Адрес для прослушивания    |
| `log_level` | `LOG_LEVEL`| `info`          | `debug` / `info` / `warn` |
| `audit_log` | `AUDIT_LOG`| `audit.jsonl`   | Путь к аудит-логу (JSONL)  |

Сертификаты в конфиге не указываются — сервис работает с сертификатами, установленными в Windows Certificate Store, через Windows API (`crypt32.dll`).

Пример конфига: [`config/example.yml`](config/example.yml).

## gRPC-контракт

```protobuf
service Signer {
  rpc Sign             (SignRequest)   returns (SignResponse);
  rpc Verify           (VerifyRequest) returns (VerifyResponse);
  rpc ListCertificates (Empty)         returns (CertListResponse);
}

message SignRequest {
  bytes  payload    = 1;  // сырые байты для подписи
  string thumbprint = 2;  // SHA1 hex сертификата, без учёта регистра
  string caller_id  = 3;  // идентификатор вызывающей стороны ("edo-client", "chestnyznak")
}
```

Полная схема: [`proto/signer/signer.proto`](proto/signer/signer.proto).

После изменения `.proto` — выполните `.\scripts\generate.ps1` и закоммитьте `gen/` вместе с `.proto`.

## Аудит-лог

На каждый вызов `Sign` в файл (JSONL) добавляется запись:

```json
{"ts":"2026-04-07T12:00:00Z","caller":"edo-client","thumbprint":"AB12CD...","payload_size":1024,"ok":true}
```

## Установка как Windows-служба

```powershell
# От имени администратора
.\scripts\install-service.ps1

# С явным путём к конфигу
.\scripts\install-service.ps1 -ConfigPath C:\sign-service\config\prod.yml
```

Использует [NSSM](https://nssm.cc). Логи службы: `logs\service.log` (ротация по 10 МБ).

## Тесты

```powershell
go test ./...
```

Тесты сервера используют `stubSigner` и не требуют Windows — работают на Linux CI.

## Зависимости клиентов

`gen/` закоммичен в репозиторий — клиенты могут его вендорить без запуска `protoc`.

Импорт в клиенте:
```go
import pb "github.com/SoulStalker/sign-service/gen/signer"
```
