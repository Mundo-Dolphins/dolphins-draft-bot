# Despliegue

El bot es un unico binario sin dependencias externas. Estos pasos usan systemd en Linux; en una
Raspberry Pi basta con compilar para `arm64`.

```bash
# 1. Compilar (ajusta GOARCH: amd64 o arm64)
GOOS=linux GOARCH=arm64 go build -o build/draft-bot ./cmd/bot
GOOS=linux GOARCH=arm64 go build -o build/draft-bot-tools ./cmd/tools

# 2. Copiar al servidor (ajusta usuario y host)
ssh user@host 'sudo mkdir -p /opt/draft-bot/{config,data} && sudo chown -R user /opt/draft-bot'
scp build/draft-bot build/draft-bot-tools user@host:/opt/draft-bot/
scp config/config.yaml user@host:/opt/draft-bot/config/

# 3. Credenciales: crea /opt/draft-bot/.env en el servidor con permisos 600
ssh user@host 'install -m 600 /dev/null /opt/draft-bot/.env && editor /opt/draft-bot/.env'

# 4. Servicio
scp deploy/draft-bot.service.example user@host:/tmp/draft-bot.service
ssh user@host 'sudo mv /tmp/draft-bot.service /etc/systemd/system/ && sudo systemctl daemon-reload && sudo systemctl enable --now draft-bot'
```

Antes del draft, valida la configuracion y envia un mensaje de prueba:

```bash
/opt/draft-bot/draft-bot-tools validate-config
/opt/draft-bot/draft-bot-tools send-test-message
```

No subas nunca `.env` ni `data/state.json` a un repositorio. Si quieres versionar tu configuracion
real (ventanas, picks, canal), hazlo en un repositorio **privado** aparte.
