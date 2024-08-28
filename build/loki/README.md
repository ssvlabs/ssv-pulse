# Loki playground

This docker-compose file will start a Loki stack with Grafana so that we can use SSV Pulse to send SSV logs to a local Loki instance.

## Requirements

- A folder with SSV logs
- Docker compose

## Usage

From the root of the repository, run:

```bash
make
docker compose -f build/loki/docker-compose.yml up -d
./cmd/pulse/bin/pulse send-loki --loki-url http://localhost:3100/loki/api/v1/push --folder ~/Downloads/Logs1
```

If you are wanting to process logs from different sources/partners, you can append the `--label` flag to the `send-loki` command:

```bash
./cmd/pulse/bin/pulse send-loki --loki-url http://localhost:3100/loki/api/v1/push --folder ~/Downloads/Logs1 --label partner=partner1
```

Then, you can access Grafana at `http://localhost:3000/explore` and query the logs using the preconfigured Loki datasource.


## Cleanup
To clean up, just run:

```bash
docker compose -f build/loki/docker-compose.yml down -v
````