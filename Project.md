## Project Structure

```
Infermal_v2/
├── main.go                          # Root launcher entry point
├── go.mod                           # Launcher module wiring root -> Engine
│
├── Setting/                         # Root runtime configuration
│   ├── setting.conf                 # Runtime configuration
│   ├── redis.yaml                   # Redis connection settings
│   └── root.conf                    # Root DNS hints
|
├── core/
│   ├── accelerator/         # Throughput accelerator
│   ├── adaptive/            # PID-style adaptive controller
│   ├── config/              # Config loader
│   ├── cooldown/            # Back-pressure gate
│   ├── filewriter/          # Buffered NDJSON writer
│   ├── logger/              # Structured logger
│   ├── progressBar/         # CLI progress bar
│   ├── ratelimiter/         # Redis token bucket
│   ├── redis/               # Redis client wrapper
│   ├── ui/                  # Spinner · banner · end summary
│   └── worker/              # Priority worker pool
│
├── Engine/                          # All application code and engine assets
│   ├── engine.go                    # Engine entry called by root main.go
│   │
│   ├── Input/
│   │   └── Keywords.csv             # Seed keyword list
│   │
│   └── app/
│       ├── app.go                   # Orchestration — Run(Recon+Intel+DNS)
│       ├── runtime_task.go          # Per-domain task lifecycle
│       ├── runtime_intel.go         # Intel pipeline and queue
│       ├── runtime_intel_helpers.go
│       ├── runtime_intel_resolver.go
│       ├── runtime_progress.go      # Live CLI progress rows
│       ├── runtime_tuner.go         # Adaptive controller
│       │
│       ├── DNS/
│       │   ├── dns.go               # Everything Related to DNS Interface
│       │   ├── dns_utils.go
│       │   ├── dns_fallback.go
│       │   ├── rResolver/           # Recursive resolver
│       │   └── sResolver/           # Stub resolver
│       │
│       ├── Recon/
│       │   ├── recon.go             # Recon struct
│       │   ├── recon_generator.go   # GenerateScoredDomains()
│       │   ├── recon_generator_human.go
│       │   ├── recon_generator_validate.go
│       │   ├── dga/                 # DGA algorithm implementations
│       │   └── Mutation/            # Mutation algorithm implementations
│       │
│       └── intel/
│           ├── intel.go             # DNSIntelService public API
│           └── dns_intel/      
│
├── Output/
│   ├── Generated_Domains.ndjson     # Domain permutation output
│   └── DNS_Intel.ndjson             # Behavioral fingerprint output
│
├── Logs/
│   ├── app_<timestamp>.log
│   ├── dns_<timestamp>.log
│   └── ratelimiter_<timestamp>.log
│
├── scripts/
│   └── arch_check.go                # Repo-level architecture scoring
│
├── Makefile                         # Root quality pipeline wrapper
├── README.md
├── LICENSE / NOTICE
└── System_Design/
    ├── Infermal_v2-SAD.drawio.png
    ├── Infermal_v2-DataFlow-Part1.png
    └── Infermal_v2-DataFlow-Part2.png
```
