// Package dolphinsdraftbot is the module root of dolphins-draft-bot, a Telegram
// bot that watches the NFL Draft and announces selections and trades that
// involve one team (Miami Dolphins by default).
//
// The module is organised as a hexagonal application whose packages can be
// imported individually:
//
//   - domain: draft models and the rules that detect selections and trades
//   - ports: interfaces implemented by the adapters
//   - application: the use case, the polling scheduler and message formatting
//   - adapters/outbound/espn, telegram, storage: ESPN reader, Telegram
//     publisher and JSON event store
//   - adapters/inbound/runner: wiring of the pieces above (runner.Run)
//   - platform/config, platform/logging: YAML + .env configuration and slog setup
//
// Until v1.0.0 the API may change between minor versions.
package dolphinsdraftbot
