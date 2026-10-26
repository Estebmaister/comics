# Architecture Improvement Plans

These plans implement the approved
[architecture review and hardening design](../specs/2026-10-26-architecture-review-redesign.md).
Each plan is independently reviewable and testable; execute them in the order
listed unless a plan's dependency section explicitly permits parallel work.

1. [Security boundary](2026-10-26-security-boundary.md)
2. [Discovery and data integrity](2026-10-26-discovery-data-integrity.md)
3. [Transactional repository and schema parity](2026-10-26-transactional-repository-schema-parity.md)
4. [Lifecycle and operations](2026-10-26-lifecycle-operations.md)
5. [Scrape reliability](2026-10-26-scrape-reliability.md)
6. [Authentication hardening](2026-10-26-auth-hardening.md)
7. [Frontend API consistency](2026-10-26-frontend-api-consistency.md)
8. [Contract and legacy cleanup](2026-10-26-contract-legacy-cleanup.md)
9. [Observability and verification](2026-10-26-observability-verification.md)

The first three plans are the critical security and data-integrity sequence.
The lifecycle, scrape, auth, and frontend plans can then proceed with their
documented dependencies. Contract cleanup follows consumer verification.
Observability requirements are embedded in every plan and completed by the
final cross-cutting verification plan.
