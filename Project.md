## Project Structure

```
Infermal_v2/
├── main.go                          # Entry point
├── go.mod / go.sum
│
├── Input/
│   └── Keywords.csv                 # Seed keyword list
│
├── Output/
│   ├── Generated_Domains.ndjson     # Domain permutation output
│   └── DNS_Intel.ndjson             # Behavioral fingerprint output
│
├── Setting/
│   ├── setting.conf                 # Runtime configuration
│   └── redis.yaml                   # Redis connection settings
│
├── Logs/
│   ├── app_<timestamp>.log
│   ├── dns_<timestamp>.log
│   └── ratelimiter_<timestamp>.log
│
├── Modules/app/
│   ├── app.go                       # Orchestration — Run()
│   ├── runtime_task.go              # Per-domain task lifecycle
│   ├── runtime_intel.go             # Intel pipeline and queue
│   ├── runtime_intel_helpers.go
│   ├── runtime_intel_resolver.go
│   ├── runtime_progress.go          # Live CLI progress rows
│   ├── runtime_tuner.go             # Adaptive controller
│   │
│   ├── DNS/
│   │   ├── dns.go                   # Engine entry point
│   │   ├── dns_utils.go
│   │   ├── dns_fallback.go
│   │   ├── rResolver/               # Recursive resolver
│   │   └── sResolver/               # Stub resolver
│   │
│   ├── Recon/
│   │   ├── recon.go                 # DNS interface + Recon struct
│   │   ├── recon_generator.go       # GenerateScoredDomains()
│   │   ├── recon_generator_human.go # Human-likeness filter
│   │   ├── recon_generator_validate.go
│   │   ├── dga/                     # DGA algorithm implementations
│   │   │   ├── bitsquatting/
│   │   │   ├── typo_squat/
│   │   │   ├── combo_squat/
│   │   │   ├── homograph/
│   │   │   ├── sound_squat/
│   │   │   ├── subdomain_squat/
│   │   │   └── jarowinkler/
│   │   └── Mutation/                # Mutation algorithm implementations
│   │       ├── character/
│   │       ├── seed/
│   │       └── hashchain/
│   │
│   ├── intel/
│   │   ├── intel.go                 # DNSIntelService public API
│   │   └── dns_intel/
│   │       └── dns_intel.go         # Processor · parallel lookups · provider extraction
│   │
│   └── core/
│       ├── accelerator/             # Throughput accelerator
│       ├── adaptive/                # PID-style adaptive controller
│       ├── config/                  # Config loader
│       ├── cooldown/                # Back-pressure gate
│       ├── filewriter/              # Buffered NDJSON writer
│       ├── logger/                  # Structured logger
│       ├── progressBar/             # CLI progress bar
│       ├── ratelimiter/             # Redis token bucket
│       ├── redis/                   # Redis client wrapper
│       ├── session/
│       ├── ui/                      # Spinner · banner · end summary
│       └── worker/                  # Priority worker pool
│
└── System Design/
    ├── Infermal_v2-SAD.drawio               # System architecture diagram
    ├── Infermal_v2-DataFlow-Part1.drawio    # Data flow: generation → resolution
    └── Infermal_v2-DataFlow-Part2.drawio    # Data flow: intel pipeline → output
```
