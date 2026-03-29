<div align="center">

# DIBs

**Domain Intelligence & Behavior System**

*DNS Intelligence & Infrastructure Behavior Analysis Framework*

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square\&logo=go\&logoColor=white)](https://golang.org/)
[![Build Status](https://img.shields.io/badge/Build-Passing-2ea44f?style=flat-square\&logo=github-actions\&logoColor=white)](https://github.com/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue?style=flat-square)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Active-brightgreen?style=flat-square)](https://github.com/)
[![Redis](https://img.shields.io/badge/Redis-Required-DC382D?style=flat-square\&logo=redis\&logoColor=white)](https://redis.io/)
[![Platform](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat-square\&logo=linux\&logoColor=black)](https://www.kernel.org/)

*High-throughput domain intelligence, DNS resolution, and behavioral analysis at scale.*

---

[Overview](#overview) · [Architecture](#architecture) · [Features](#-features) · [Getting Started](#-getting-started) · [API](#api-control-plane) · [Use Cases](#-use-cases)

</div>

---

## Overview

**DIBs** is a modular, high-performance framework for generating, resolving, analyzing, and correlating domain intelligence at scale.

It is built for offensive security workflows, threat intelligence operations, and DNS telemetry analysis.

The system operates as a unified pipeline:

```
Domain Generation → DNS Resolution → Intelligence Extraction → Correlation → Output
```

DIBs is designed to be **scalable**, **pipeline-driven**, and **SIEM-ready**, producing structured NDJSON output for downstream systems.

---

## Architecture

DIBs follows a unidirectional processing pipeline with adaptive runtime control.

### High-Level Diagram

> Placeholder — replace with actual architecture diagram

```text
[ Diagram Placeholder ]
Domain Generation
        ↓
DNS Resolution
        ↓
Intelligence Extraction
        ↓
Correlation Engine
        ↓
NDJSON Output
```

---

## 🚀 Features

### Domain Mutation Engine

* Bitsquatting
* Typosquatting
* Combosquatting
* Homograph attacks
* Phonetic mutations
* Similarity scoring (Jaro–Winkler)
* Subdomain permutations

All generated domains are validated, deduplicated, and scored.

---

### High-Speed DNS Engine

* Recursive + Stub resolution modes
* Multi-record support (A, AAAA, CNAME, MX, TXT, SOA)
* Adaptive timeout and retry logic
* High-concurrency worker pool

---

### Intelligence Extraction

* A / AAAA enumeration
* CNAME chain mapping
* Nameserver profiling
* MX / TXT extraction
* Provider attribution
* TTL anomaly detection
* Fast-flux detection
* DNSSEC validation

---

### Correlation Engine

* Infrastructure clustering (IP / ASN)
* Domain relationship mapping
* Shared infrastructure detection

---

### Output System

All outputs are NDJSON-based:

* Generated domains
* Resolved domains
* DNS intelligence records
* Infrastructure clusters
* Runtime metrics

---

## ⚙️ Getting Started

### Prerequisites

* Go 1.21+
* Redis 6.0+
* Linux (recommended)

---

### Installation

```bash
git clone https://github.com/<your-username>/DIBs.git
cd DIBs
go mod tidy
```

---

### Run

```bash
go run .
```

---

## API Control Plane

DIBs exposes an API for scan lifecycle control.

### Core Endpoints

| Endpoint        | Method |
| --------------- | ------ |
| /healthz        | GET    |
| /api/v3/start   | POST   |
| /api/v3/stop    | POST   |
| /api/v3/status  | GET    |
| /api/v3/metrics | GET    |

Authentication uses **ed25519 public key validation** via `X-API-Key`.

---

## 🎯 Use Cases

* Threat intelligence enrichment
* Phishing infrastructure detection
* Red team reconnaissance
* Domain monitoring
* Incident response
* DNS telemetry research

---

## License

Apache License 2.0

---

## Author

Biswadeb Mukherjee

Offensive Security Specialist · Malware Engineer

---

<div align="center">
<sub>Built for operators who need answers, not dashboards.</sub>
</div>
