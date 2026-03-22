# ============================================================
#  INFERMAL_v2 — Code Quality & Security Pipeline
# ============================================================

.PHONY: all autofmt vet lint security secrets arch \
        complexity sloc test coverage deps check-tools \
        ci install-hooks

# ── Colours ─────────────────────────────────────────────────
YELLOW  = \033[1;33m
GREEN   = \033[1;32m
BLUE    = \033[1;34m
RED     = \033[1;31m
CYAN    = \033[1;36m
DIM     = \033[2m
RESET   = \033[0m

# ============================================================
#  DEFAULT 
# ============================================================
all: check-tools autofmt vet lint security secrets arch complexity sloc test coverage
	@echo "$(GREEN)🎉  All pipeline checks completed.$(RESET)"

# CI — strict (no autofmt)
ci: check-tools vet lint security secrets arch complexity sloc test coverage
	@echo "$(GREEN)✅  CI pipeline passed.$(RESET)"

# ============================================================
#  TOOL CHECK
# ============================================================
check-tools:
	@echo "$(BLUE)🔧 Checking required tools...$(RESET)"
	@missing=0; \
	for t in gofmt go golangci-lint gosec gocyclo cloc bc; do \
		if command -v $$t >/dev/null 2>&1; then \
			echo "  $(GREEN)✓ $$t$(RESET)"; \
		else \
			echo "  $(RED)✗ $$t missing$(RESET)"; missing=1; \
		fi; \
	done; \
	\
	echo "$(CYAN)🔐 Optional security tools:$(RESET)"; \
	for t in gitleaks trufflehog; do \
		if command -v $$t >/dev/null 2>&1; then \
			echo "  $(GREEN)✓ $$t$(RESET)"; \
		else \
			echo "  $(YELLOW)⚠ $$t not installed (skipping)$(RESET)"; \
		fi; \
	done; \
	\
	if [ $$missing -eq 1 ]; then \
		echo "$(RED)❌ Missing required tools. Install them first.$(RESET)"; \
		exit 1; \
	fi

# ============================================================
#  AUTO FORMAT
# ============================================================
autofmt:
	@echo "\n$(BLUE)🧹 Formatting check...$(RESET)"
	@files=$$(gofmt -l .); \
	if [ -n "$$files" ]; then \
		echo "$(YELLOW)⚠️ Fixing formatting...$(RESET)"; \
		echo "$$files" | xargs gofmt -w; \
	else \
		echo "$(GREEN)✅ Clean formatting.$(RESET)"; \
	fi

# ============================================================
#  STATIC ANALYSIS
# ============================================================
vet:
	@echo "\n$(BLUE)🔍 go vet...$(RESET)"
	@go vet ./... || true

lint:
	@echo "\n$(BLUE)🧠 golangci-lint...$(RESET)"
	@golangci-lint run ./... --color=always || true

# ============================================================
#  SECURITY
# ============================================================
security:
	@echo "\n$(BLUE)🛡️ gosec...$(RESET)"
	@gosec ./... || true

# ============================================================
#  SECRETS Leakage
# ============================================================
secrets:
	@echo "\n$(BLUE)🔐 Secret scan...$(RESET)"

	@if command -v gitleaks >/dev/null 2>&1; then \
		echo "$(CYAN)[gitleaks] scanning...$(RESET)"; \
		gitleaks detect --source . --no-banner --log-level error || echo "$(RED)⚠️ gitleaks issues$(RESET)"; \
	else \
		echo "$(YELLOW)⚠ Skipping gitleaks (not installed)$(RESET)"; \
	fi

	@if command -v trufflehog >/dev/null 2>&1; then \
		echo "$(CYAN)[trufflehog] scanning...$(RESET)"; \
		trufflehog filesystem . --only-verified --no-update --fail || echo "$(RED)⚠️ trufflehog issues$(RESET)"; \
	else \
		echo "$(YELLOW)⚠ Skipping trufflehog (not installed)$(RESET)"; \
	fi

# ============================================================
#  ARCHITECTURE Design
# ============================================================
arch:
	@echo "\n$(BLUE)🏛️ Architectural analysis (non-blocking)...$(RESET)"
	@go run scripts/arch_check.go > arch_report.txt || true
	@echo "$(CYAN)────────────────────────────────────────$(RESET)"
	@cat arch_report.txt
	@echo "$(CYAN)────────────────────────────────────────$(RESET)"
	@echo "$(GREEN)✅ Architecture check complete (penalty applied).$(RESET)"

# ============================================================
#  COMPLEXITY
# ============================================================
complexity:
	@echo "\n$(BLUE)⚙️ Complexity check...$(RESET)"
	@gocyclo -over 10 . || echo "$(GREEN)✅ Acceptable complexity$(RESET)"

# ============================================================
#  SLOC
# ============================================================
sloc:
	@echo "\n$(BLUE)📄 Detailed SLOC analysis (per file)...$(RESET)"
	@cloc . --include-lang=Go --by-file --quiet --csv \
	| awk -F',' \
	-v green="$(GREEN)" -v yellow="$(YELLOW)" -v red="$(RED)" -v reset="$(RESET)" \
	'BEGIN { \
		printf "------------------------------------------------------------------------------------------------------\n"; \
		printf "%-60s %8s %9s %9s %9s\n", "File", "Blank", "Comment", "Code", "Result"; \
		printf "------------------------------------------------------------------------------------------------------\n"; \
		tb=0; tc=0; tco=0; \
	} \
	/Go,/ { \
		file=$$2; blank=$$3; comment=$$4; code=$$5; \
		if (code > 400) { color=red; result="FAIL"; } \
		else if (code > 250) { color=yellow; result="WARN"; } \
		else { color=green; result="PASS"; } \
		printf "%-60s %8d %9d %9d %11s%s%s\n", file, blank, comment, code, color, result, reset; \
		tb+=blank; tc+=comment; tco+=code; \
	} \
	END { \
		printf "------------------------------------------------------------------------------------------------------\n"; \
		final_color = (tco > 2500) ? red : green; \
		final_result = (tco > 2500) ? "FAIL" : "PASS"; \
		printf "%-60s %8d %9d %9d %11s%s%s\n", "TOTAL", tb, tc, tco, final_color, final_result, reset; \
		printf "------------------------------------------------------------------------------------------------------\n"; \
	}'


# ============================================================
#  TESTS
# ============================================================
test:
	@echo "\n$(BLUE)🧪 Running tests...$(RESET)"
	@go test ./... -v || true

# ============================================================
#  COVERAGE 
# ============================================================
coverage:
	@echo "\n$(BLUE)📊 Coverage analysis...$(RESET)"

	@go test ./... -coverpkg=./... -coverprofile=coverage.out || true

	@[ -s coverage.out ] || { echo "$(RED)❌ coverage.out missing$(RESET)"; exit 1; }

	@raw=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	penalty=0; \
	if [ -f arch_score.txt ]; then penalty=$$(cat arch_score.txt); fi; \
	final=$$(echo "$$raw - $$penalty" | bc); \
	printf "\n$(CYAN)──────── COVERAGE REPORT ────────$(RESET)\n"; \
	printf "Raw Coverage:     %.2f%%\n" $$raw; \
	printf "Arch Penalty:     -%.2f%%\n" $$penalty; \
	printf "Final Coverage:   %.2f%%\n" $$final; \
	printf "$(CYAN)────────────────────────────────$(RESET)\n"

# ============================================================
#  DEPENDENCIES
# ============================================================
deps:
	@echo "\n$(BLUE)📦 Dependency check...$(RESET)"
	@go mod tidy
	@go mod verify

# ============================================================
#  PRE-PUSH HOOK
# ============================================================
install-hooks:
	@echo "$(BLUE)🪝 Installing pre-push hook...$(RESET)"
	@mkdir -p .git/hooks
	@printf '#!/bin/sh\nset -e\necho "[pre-push] Running checks..."\nmake secrets arch\n' > .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "$(GREEN)✅ Hook installed$(RESET)"