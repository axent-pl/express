PWD := $(dir $(abspath $(firstword $(MAKEFILE_LIST))))
REPORT_DIR := test/reports

.PHONY: sca sca-lint sca-gosec sca-govulncheck

.IGNORE: sca sca-lint sca-gosec sca-govulncheck

sca-lint:
	@mkdir -p $(REPORT_DIR)
	@docker run --rm -v "$(PWD)":/app -w /app golangci/golangci-lint:latest \
		golangci-lint run > $(REPORT_DIR)/golangci-lint.txt
	@echo "SCA golangci-lint completed"

sca-gosec:
	@mkdir -p $(REPORT_DIR)
	@docker run --rm -it -v "$(PWD)":/workspace -w /workspace securego/gosec:2.24.6 -out $(REPORT_DIR)/gosec.txt ./...
	@echo "SCA gosec completed"

sca-govulncheck:
	@mkdir -p $(REPORT_DIR)
	@docker run --rm -v "$(PWD)":/app -w /app golang:1.25.7 go mod download && go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./... >$(REPORT_DIR)/govulncheck.txt
	@echo "SCA govulncheck completed"

sca: sca-lint sca-gosec sca-govulncheck
	@echo "SCA completed"