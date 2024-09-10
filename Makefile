VERSION = sha-$(shell git rev-parse HEAD)

# Start up the Looking Glass Consumer
.PHONY: rebuild-lg-consumer
rebuild-lg-consumer:
	@echo "Starting up the Looking Glass Consumer"
	@VERSION=${VERSION} docker compose -f cmd/stream/docker-compose.yml up -d --build

.PHONY: lg-consumer-up
lg-consumer-up:
	@echo "Starting up the Looking Glass Consumer"
	@VERSION=${VERSION} docker compose -f cmd/stream/docker-compose.yml up -d

.PHONY: lg-consumer-down
lg-consumer-down:
	@echo "Shutting down the Looking Glass Consumer"
	@VERSION=${VERSION} docker compose -f cmd/stream/docker-compose.yml down

.PHONY: plc-exporter-up
plc-exporter-up:
	@echo "Starting up the PLC Exporter"
	@VERSION=${VERSION} docker compose -f cmd/plc/docker-compose.yml up -d --build

.PHONY: plc-exporter-down
plc-exporter-down:
	@echo "Shutting down the PLC Exporter"
	@VERSION=${VERSION} docker compose -f cmd/plc/docker-compose.yml down
