# AGENTS.md

## Coding Rules for Infermal_v2

1. Every source file must stay between 150 LOC and 250 LOC with extra 50 LOC as a grace means code file shouldn't exceed 300 LOC.
2. Every function must stay under 40 LOC and must be split into smaller helpers if it grows.
3. Each package must have a single clear responsibility and must not mix unrelated logic.
4. Prefer modifying existing modules instead of creating new files unless responsibility clearly differs.
5. Never create generic utility packages that mix multiple concerns.

## DNS & Networking

6. All DNS operations must be context-aware with proper timeout and cancellation.
7. DNS resolver must follow adaptive fallback: UDP → System → future extensions (DoT/DoH).
8. Never assume port 53 is available; all network logic must be environment-adaptive.
9. DNS resolution must never fail silently and must always return or log errors.
10. Resolver fallback logic must remain inside the DNS module only.

## Concurrency & Workers

11. All long-running operations must use context.Context.
12. Worker tasks must be idempotent and safe for retries.
13. Channel operations must never block indefinitely and must respect context cancellation.
14. Avoid shared mutable state across workers unless properly synchronized.
15. Prefer stateless workers with external state handled via Redis.

## Architecture

16. Maintain strict dependency flow: main → app → recon → DNS → core.
17. Upward imports are forbidden.
18. Horizontal dependencies between modules are forbidden.
19. Core infrastructure packages must never import recon or app modules.
20. Circular dependencies are strictly forbidden.

## Interfaces & Decoupling

21. Define interfaces in the consuming module and implement them in the provider module.
22. Do not directly bind implementations inside modules.
23. All implementations must be wired through the application layer.

## File I/O

24. File writing must be buffered, batched, and safe on shutdown.
25. Writers must support periodic flush and graceful close.
26. Data loss on interrupt must be prevented via proper shutdown handling.

## Redis & Caching

27. Redis must be treated as a cache layer, not a source of truth.
28. Cache failures must not block execution.
29. All Redis entries must use TTL.
30. Redis errors must be handled gracefully without panic.

## Error Handling & Logging

31. Errors must never be ignored.
32. Errors must be logged or propagated with proper context.
33. Avoid excessive logging inside tight loops unless debugging.

## Code Style

34. Use clear and descriptive naming following Go conventions.
35. Prefer composition and small structs over complex hierarchies.
36. Avoid unnecessary abstractions.
37. Keep public APIs minimal and focused.
38. Do not expose internal implementation details unnecessarily.

## Performance & Stability

39. Prioritize correctness, then stability, then performance.
40. Avoid premature optimization.
41. Use batching and pooling only when necessary and justified.
42. All async systems must support backpressure and safe shutdown.

## Reliability

43. Never assume network or DNS reliability.
44. All external interactions must include timeout and controlled retry logic.

## Evolution

45. New features must respect existing architecture and module boundaries.
46. Avoid rewriting stable modules unless necessary.
47. Maintain backward compatibility for configs and I/O formats.

## Adaptive Resolver Requirement

48. Resolver must dynamically adapt to environment and automatically switch fallback modes.
