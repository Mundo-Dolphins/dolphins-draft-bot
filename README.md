# dolphins-draft-bot

[![CI](https://github.com/Mundo-Dolphins/dolphins-draft-bot/actions/workflows/ci.yml/badge.svg)](https://github.com/Mundo-Dolphins/dolphins-draft-bot/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Bot en Go que vigila el NFL Draft y publica en Telegram los eventos que afectan a un equipo
(por defecto **Miami Dolphins**, Draft 2026):

- **Selección**: el equipo ha elegido a un jugador (`selection`).
- **Traspaso saliente**: un pick original del equipo ha pasado a otro (`trade_out`).
- **Traspaso entrante**: el equipo ha adquirido un pick que no era suyo (`trade_in`).

Cada evento se publica una sola vez (idempotencia por `event_key`, persistida en un fichero JSON).
Los mensajes están en español.

> Proyecto de la comunidad [Mundo Dolphins](https://mundodolphins.es). No está afiliado ni
> respaldado por la NFL, ESPN ni Miami Dolphins; los nombres se usan solo de forma descriptiva.

## Cómo funciona

1. Un *scheduler* duerme hasta la próxima ventana del draft (configurable, con zona horaria).
2. Durante la ventana consulta el feed del draft de ESPN con un intervalo por ronda.
3. Un parser tolerante normaliza la respuesta (hay dos formatos conocidos) a un `DraftSnapshot`.
4. Las reglas de dominio detectan selecciones y traspasos y se publican en Telegram.

> **Aviso:** el bot usa el endpoint público **no documentado** de ESPN
> (`site.api.espn.com/apis/site/v2/sports/football/nfl/draft`). Puede cambiar o dejar de funcionar
> sin previo aviso. Respeta los intervalos de polling razonables (los por defecto son 25–60 s).

## Arquitectura

Hexagonal (puertos y adaptadores):

- `cmd/bot`: binario principal con el scheduler.
- `cmd/tools`: utilidades (`validate-config`, `send-test-message`, `print-state`, `simulate-payload`, `parse-fixture`).
- `domain`: modelos y reglas de negocio.
- `application`: caso de uso, scheduler y formateo de mensajes.
- `ports`: interfaces.
- `adapters/outbound/{espn,telegram,storage}`: ESPN (HTTP), Telegram (HTTP), persistencia JSON.
- `adapters/inbound/runner`: composición completa (`runner.Run`) y reloj.
- `platform/{config,logging}`: configuración (YAML + `.env`) y logging (`slog`).

## Uso como librería

Todos los paquetes son importables (no hay `internal/`). El caso más simple es reutilizar la
composición completa y aportar tu propia configuración y despliegue:

```go
package main

import (
	"fmt"
	"os"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/adapters/inbound/runner"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/platform/config"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/platform/logging"
)

func main() {
	cfg, err := config.Load("config/config.yaml", ".env")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger := logging.New(cfg.Logging.Level, cfg.Logging.Format)
	if err := runner.Run(cfg, logger); err != nil {
		logger.Error("bot finished with error", "error", err)
		os.Exit(1)
	}
}
```

Para algo más a medida, `application.NewDraftService` y `application.NewScheduler` aceptan
cualquier implementación de las interfaces de `ports` (por ejemplo, otro publicador distinto de
Telegram).

> **Estabilidad:** mientras la versión sea `v0.x`, la API puede cambiar entre versiones menores.
> Fija una versión concreta en tu `go.mod`.

## Inicio rápido

Requisitos: Go 1.22 o posterior y un bot de Telegram (créalo con [@BotFather](https://t.me/BotFather)).

```bash
git clone https://github.com/Mundo-Dolphins/dolphins-draft-bot.git
cd dolphins-draft-bot

cp .env.example .env        # rellena TELEGRAM_BOT_TOKEN y TELEGRAM_CHAT_ID
go run ./cmd/tools validate-config
go run ./cmd/tools send-test-message
go run ./cmd/bot -config config/config.yaml -env-file .env
```

Con `app.dry_run: true` en el YAML el bot registra los mensajes en el log en lugar de enviarlos.

## Configuración

- **YAML** (`config/config.yaml`): todo lo no sensible: año, URL de ESPN, equipo, picks originales,
  ventanas del draft, intervalos de polling, HTTP y almacenamiento.
- **`.env`** o variables de entorno: `TELEGRAM_BOT_TOKEN` y `TELEGRAM_CHAT_ID`. Los valores del
  `.env` tienen prioridad.

Para otro equipo cambia `draft.miami_team_id`, `draft.miami_team_abbr`,
`draft.miami_display_name` (también se usa en los mensajes) y `draft.original_miami_picks`. Los ids
de equipo son los de ESPN.

La configuración incluida es un ejemplo funcional del Draft 2026. Si tu configuración real contiene
datos que no quieres publicar, guárdala en un repositorio privado y pásala con `-config`.

## Desarrollo

```bash
make test        # tests (sin red real: httptest, fakes y fixtures)
make coverage    # falla si la cobertura total (sin `cmd/`) es < 85 %
make lint        # golangci-lint (versión fijada en el Makefile)
make ci          # fmt-check + test + coverage + lint

go run ./cmd/tools parse-fixture test/fixtures/draft_payload.json
```

## Despliegue

Ver [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) y la unidad de ejemplo
[deploy/draft-bot.service.example](deploy/draft-bot.service.example).

## Seguridad

Ver [SECURITY.md](SECURITY.md). Nunca subas `.env` ni `data/state.json`.

## Licencia

[MIT](LICENSE)
