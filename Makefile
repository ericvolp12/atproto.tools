# Start up the Looking Glass Consumer
.PHONY: lg-consumer-up
lg-consumer-up:
	@echo "Starting up the Looking Glass Consumer"
	@docker-compose -f docker-compose.yml up -d --build

.PHONY: lg-consumer-down
lg-consumer-down:
	@echo "Shutting down the Looking Glass Consumer"
	@docker-compose -f docker-compose.yml down
