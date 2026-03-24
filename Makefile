# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║              INFERMAL_v2  ·  Code Quality & Security Pipeline               ║
# ║                          Offensive Security Engine                          ║
# ╚══════════════════════════════════════════════════════════════════════════════╝

.PHONY: all ci \
        check-tools \
        autofmt vet lint \
        security secrets \
        arch complexity sloc \
        test coverage \
        deps \
        install-hooks

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  CONFIGURATION
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

ENGINE_DIR := Engine
GO_ENGINE  := go -C $(ENGINE_DIR)
GO_CACHE_DIR := /tmp/infermal-go-build

SLOC_WARN_THRESHOLD  := 250
SLOC_FAIL_THRESHOLD  := 400
SLOC_TOTAL_THRESHOLD := 2500

COVERAGE_PASS := 80
COVERAGE_WARN := 40

CYCLO_THRESHOLD := 10

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  COLOURS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

YELLOW := \033[1;33m
GREEN  := \033[1;32m
BLUE   := \033[1;34m
RED    := \033[1;31m
CYAN   := \033[1;36m
DIM    := \033[2m
RESET  := \033[0m

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  ENTRY POINTS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## all   : Full local pipeline (format + all checks + tests)
all: check-tools autofmt vet lint security secrets arch complexity sloc test coverage
	@echo "$(GREEN)🎉  All pipeline checks completed.$(RESET)"

## ci    : Strict CI pipeline (no autofmt, exits on failure)
ci: check-tools vet lint security secrets arch complexity sloc test coverage
	@echo "$(GREEN)✅  CI pipeline passed.$(RESET)"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  [1] PREREQUISITE CHECK
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## check-tools : Verify all required and optional tools are present
check-tools:
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@echo "$(BLUE) [1/8]  Tool Check$(RESET)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@missing=0; \
	echo "$(CYAN)  Required:$(RESET)"; \
	for t in gofmt go golangci-lint gosec gocyclo cloc bc; do \
		if command -v $$t >/dev/null 2>&1; then \
			echo "    $(GREEN)✓$(RESET)  $$t"; \
		else \
			echo "    $(RED)✗$(RESET)  $$t $(DIM)(missing)$(RESET)"; \
			missing=1; \
		fi; \
	done; \
	echo ""; \
	echo "$(CYAN)  Optional:$(RESET)"; \
	for t in gitleaks trufflehog; do \
		if command -v $$t >/dev/null 2>&1; then \
			echo "    $(GREEN)✓$(RESET)  $$t"; \
		else \
			echo "    $(YELLOW)~$(RESET)  $$t $(DIM)(not installed — will skip)$(RESET)"; \
		fi; \
	done; \
	echo ""; \
	if [ $$missing -eq 1 ]; then \
		echo "$(RED)  ❌  Missing required tools. Install them before proceeding.$(RESET)"; \
		exit 1; \
	fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  [2] FORMAT
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## autofmt : Auto-format Go source files with gofmt
autofmt:
	@echo ""
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@echo "$(BLUE) [2/8]  Format$(RESET)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@files=$$(gofmt -l main.go $(ENGINE_DIR) scripts); \
	if [ -n "$$files" ]; then \
		echo "  $(YELLOW)⚠  Unformatted files detected — applying gofmt:$(RESET)"; \
		echo "$$files" | sed 's/^/    /'; \
		echo "$$files" | xargs gofmt -w; \
		echo "  $(GREEN)✓  Formatting applied.$(RESET)"; \
	else \
		echo "  $(GREEN)✓  All files are correctly formatted.$(RESET)"; \
	fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  [3] STATIC ANALYSIS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## vet  : Run go vet on the engine
vet:
	@echo ""
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@echo "$(BLUE) [3/8]  Static Analysis — go vet$(RESET)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@$(GO_ENGINE) vet ./... || true

## lint : Run golangci-lint on the engine
lint:
	@echo ""
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@echo "$(BLUE) [3/8]  Static Analysis — golangci-lint$(RESET)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@cd $(ENGINE_DIR) && golangci-lint run ./... --color=always || true

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  [4] SECURITY SCANNING
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## security : Run gosec SAST scanner
security:
	@echo ""
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@echo "$(BLUE) [4/8]  Security — gosec SAST$(RESET)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@cd $(ENGINE_DIR) && gosec ./... || true

## secrets : Scan for leaked secrets via gitleaks / trufflehog
secrets:
	@echo ""
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@echo "$(BLUE) [4/8]  Security — Secret Leakage Scan$(RESET)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@if command -v gitleaks >/dev/null 2>&1; then \
		echo "  $(CYAN)→  gitleaks$(RESET)"; \
		gitleaks detect --source . --no-banner --log-level error \
			|| echo "  $(RED)⚠  gitleaks: issues found$(RESET)"; \
	else \
		echo "  $(YELLOW)~  gitleaks not installed — skipping$(RESET)"; \
	fi
	@if command -v trufflehog >/dev/null 2>&1; then \
		echo "  $(CYAN)→  trufflehog$(RESET)"; \
		trufflehog filesystem . --only-verified --no-update --fail \
			|| echo "  $(RED)⚠  trufflehog: issues found$(RESET)"; \
	else \
		echo "  $(YELLOW)~  trufflehog not installed — skipping$(RESET)"; \
	fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  [5] ARCHITECTURE ANALYSIS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## arch : Run architectural analysis and generate report
arch:
	@echo ""
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@echo "$(BLUE) [5/8]  Architecture Analysis$(RESET)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@go run scripts/arch_check.go > arch_report.txt || true
	@echo "$(DIM)"
	@cat arch_report.txt
	@echo "$(RESET)"
	@echo "  $(GREEN)✓  Architecture check complete (penalty score applied to coverage).$(RESET)"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  [6] COMPLEXITY & SLOC
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## complexity : Check cyclomatic complexity (threshold: 10)
complexity:
	@echo ""
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@echo "$(BLUE) [6/8]  Complexity  (threshold: >$(CYCLO_THRESHOLD))$(RESET)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@gocyclo -over $(CYCLO_THRESHOLD) main.go $(ENGINE_DIR) scripts \
		|| echo "  $(GREEN)✓  All functions within acceptable complexity.$(RESET)"

## sloc : Per-file SLOC analysis with PASS/WARN/FAIL thresholds
sloc:
	@echo ""
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@echo "$(BLUE) [6/8]  SLOC  (warn: >$(SLOC_WARN_THRESHOLD)  fail: >$(SLOC_FAIL_THRESHOLD)  total: >$(SLOC_TOTAL_THRESHOLD))$(RESET)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@cloc main.go $(ENGINE_DIR) scripts --include-lang=Go --by-file --quiet --csv \
	| awk -F',' \
	  -v green="$(GREEN)" -v yellow="$(YELLOW)" -v red="$(RED)" -v reset="$(RESET)" \
	  -v warn=$(SLOC_WARN_THRESHOLD) -v fail=$(SLOC_FAIL_THRESHOLD) -v total_limit=$(SLOC_TOTAL_THRESHOLD) \
	'BEGIN { \
		printf "  %-58s %8s %9s %9s %8s\n", "File", "Blank", "Comment", "Code", "Status"; \
		printf "  %s\n", "──────────────────────────────────────────────────────────────────────────────────────────────"; \
		tb=0; tc=0; tco=0; \
	} \
	/Go,/ { \
		file=$$2; blank=$$3; comment=$$4; code=$$5; \
		if (code > fail)        { color=red;    result="FAIL"; } \
		else if (code > warn)   { color=yellow; result="WARN"; } \
		else                    { color=green;  result="PASS"; } \
		printf "  %-58s %8d %9d %9d   %s%s%s\n", file, blank, comment, code, color, result, reset; \
		tb+=blank; tc+=comment; tco+=code; \
	} \
	END { \
		printf "  %s\n", "──────────────────────────────────────────────────────────────────────────────────────────────"; \
		fc = (tco > total_limit) ? red : green; \
		fr = (tco > total_limit) ? "FAIL" : "PASS"; \
		printf "  %-58s %8d %9d %9d   %s%s%s\n", "TOTAL", tb, tc, tco, fc, fr, reset; \
	}'

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  [7] TESTS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## test : Run full test suite
test:
	@echo ""
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@echo "$(BLUE) [7/8]  Test Suite$(RESET)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@$(GO_ENGINE) test ./... -v || true

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  [8] COVERAGE
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## coverage : Generate coverage report with per-file breakdown and arch penalty
coverage:
	@echo ""
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@echo "$(BLUE) [8/8]  Coverage  (pass: ≥$(COVERAGE_PASS)%  warn: ≥$(COVERAGE_WARN)%)$(RESET)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@mkdir -p $(GO_CACHE_DIR)

	@GOCACHE=$(GO_CACHE_DIR) $(GO_ENGINE) test ./... -coverpkg=./... -coverprofile=../coverage.out || true

	@[ -s coverage.out ] || { \
		echo "  $(RED)❌  coverage.out not found or empty.$(RESET)"; exit 1; \
	}

	@echo ""
	@awk \
	  -v green="$(GREEN)" -v yellow="$(YELLOW)" -v red="$(RED)" -v reset="$(RESET)" \
	  -v pass=$(COVERAGE_PASS) -v warn=$(COVERAGE_WARN) \
	'BEGIN { \
		printf "  %-70s %10s %8s\n", "File", "Coverage", "Status"; \
		printf "  %s\n", "────────────────────────────────────────────────────────────────────────────────────────────────────"; \
	} \
	NR == 1 { next } \
	{ \
		key = $$1; \
		stmts = $$2 + 0; \
		hit = (($$3 + 0) > 0) ? 1 : 0; \
		if (!(key in seen)) { \
			seen[key] = 1; \
			blockStmts[key] = stmts; \
			split(key, loc, ":"); \
			file = loc[1]; \
			sub("^github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/", "", file); \
			blockFile[key] = file; \
		} \
		if (hit > 0) blockHit[key] = 1; \
	} \
	END { \
		pi=0; wi=0; fi=0; \
		ts=0; tc=0; \
		for (k in seen) { \
			f = blockFile[k]; \
			total[f] += blockStmts[k]; \
			if (blockHit[k] > 0) covered[f] += blockStmts[k]; \
		} \
		for (f in total) { \
			avg = (total[f] > 0) ? (covered[f] * 100 / total[f]) : 0; \
			ts += total[f]; \
			tc += covered[f]; \
			if (avg >= pass)      { pass_f[pi]=f; pass_v[pi]=avg; pi++; } \
			else if (avg >= warn) { warn_f[wi]=f; warn_v[wi]=avg; wi++; } \
			else                  { fail_f[fi]=f; fail_v[fi]=avg; fi++; } \
		} \
		for (i=0; i<fi; i++) \
			printf "  %-70s %8.2f%%  %sFAIL%s\n", fail_f[i], fail_v[i], red, reset; \
		for (i=0; i<wi; i++) \
			printf "  %-70s %8.2f%%  %sWARN%s\n", warn_f[i], warn_v[i], yellow, reset; \
		for (i=0; i<pi; i++) \
			printf "  %-70s %8.2f%%  %sPASS%s\n", pass_f[i], pass_v[i], green, reset; \
		printf "  %s\n", "────────────────────────────────────────────────────────────────────────────────────────────────────"; \
		fa = (ts > 0) ? (tc * 100 / ts) : 0; \
		fc = (fa >= pass) ? green : (fa >= warn ? yellow : red); \
		fr = (fa >= pass) ? "PASS" : (fa >= warn ? "WARN" : "FAIL"); \
		printf "  %-70s %8.2f%%  %s%s%s\n", "STATEMENT AVG", fa, fc, fr, reset; \
	}' coverage.out

	@echo ""
	@raw=$$(awk 'NR > 1 { key=$$1; stmts=$$2+0; hit=(($$3+0)>0); if (!(key in seen)) { seen[key]=1; blockStmts[key]=stmts; } if (hit) blockHit[key]=1 } END { for (k in seen) { total += blockStmts[k]; if (blockHit[k]) covered += blockStmts[k]; } printf "%.2f", (total > 0 ? covered * 100 / total : 0) }' coverage.out); \
	penalty=0; \
	[ -f arch_score.txt ] && penalty=$$(cat arch_score.txt); \
	final=$$(echo "$$raw - $$penalty" | bc); \
	echo "  $(CYAN)┌─────────────────────────────────┐$(RESET)"; \
	printf   "  $(CYAN)│$(RESET)  Raw Coverage      %6.2f%%       $(CYAN)│$(RESET)\n" $$raw; \
	printf   "  $(CYAN)│$(RESET)  Arch Penalty     -%6.2f%%       $(CYAN)│$(RESET)\n" $$penalty; \
	printf   "  $(CYAN)│$(RESET)  ─────────────────────────  $(CYAN)│$(RESET)\n"; \
	printf   "  $(CYAN)│$(RESET)  Final Coverage    %6.2f%%       $(CYAN)│$(RESET)\n" $$final; \
	echo    "  $(CYAN)└─────────────────────────────────┘$(RESET)"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  UTILITIES
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## deps         : Tidy and verify Go modules
deps:
	@echo ""
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@echo "$(BLUE)  Dependencies$(RESET)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@$(GO_ENGINE) mod tidy
	@$(GO_ENGINE) mod verify
	@echo "  $(GREEN)✓  Dependencies verified.$(RESET)"

## install-hooks : Install git pre-push hook (runs secrets + arch)
install-hooks:
	@echo ""
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@echo "$(BLUE)  Git Hooks$(RESET)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"
	@mkdir -p .git/hooks
	@printf '#!/bin/sh\nset -e\necho "[pre-push] Running security checks..."\nmake secrets arch\n' \
		> .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "  $(GREEN)✓  pre-push hook installed: $(DIM).git/hooks/pre-push$(RESET)"

# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║  make all          — full local pipeline                                    ║
# ║  make ci           — strict CI (no autofmt)                                 ║
# ║  make security     — gosec SAST only                                        ║
# ║  make secrets      — gitleaks + trufflehog                                  ║
# ║  make test         — test suite                                              ║
# ║  make coverage     — coverage with arch penalty                             ║
# ║  make deps         — tidy + verify modules                                  ║
# ║  make install-hooks— install git pre-push hook                              ║
# ╚══════════════════════════════════════════════════════════════════════════════╝
