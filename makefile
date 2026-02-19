PWD := $(dir $(abspath $(firstword $(MAKEFILE_LIST))))
REPORT_DIR := test/reports

.PHONY: lint

.IGNORE: lint

lint:
	@mkdir -p $(REPORT_DIR)
	@docker run --rm -v "$(PWD)":/app -w /app golangci/golangci-lint:latest \
		golangci-lint run > $(REPORT_DIR)/golangci-lint.txt
	@echo "lint completed"
