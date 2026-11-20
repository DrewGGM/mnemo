# mnemo

Long-term memory for LLM agents, as a small Go library and a self-hostable service backed by PostgreSQL and nothing else.

**Status: design phase.** No application code yet. This repository currently holds the specification, the architecture, the evaluation protocol and the ordered implementation plan. Every benchmark cell below says "pending" until the harness has produced it.

## The problem

An agent forgets everything between sessions unless the application re-sends the whole history, which costs tokens on every turn and stops working once the history outgrows the context window. The obvious fix, embedding old messages and retrieving the top-k, breaks on exactly the cases that matter: facts that change ("moved from Bogotá to Medellín"), questions about when something was true, and questions whose answer is spread over several sessions.

Several mature projects already address this (Mem0, Zep/Graphiti, Letta). They are Python-first, and the graph-based ones need a dedicated graph database. Their published benchmark numbers are also hard to compare: vendors have publicly disputed each other's LoCoMo results, and an independent audit found errors in that benchmark's answer key (see [docs/evaluation-plan.md](docs/evaluation-plan.md)).

## What mnemo does

- Stores every message as an immutable **episode** (episodic memory).
- Extracts atomic **facts** from conversations (semantic memory), each linked to the episodes it came from.
- Resolves conflicts by **superseding** instead of overwriting: the old fact keeps its validity interval, so "where did I live before Medellín?" still has an answer.
- **Consolidates** (merges duplicates, summarizes old episodes) and **forgets** (decay-based archiving, plus hard deletion on request).
- Retrieves with **hybrid search**: vector similarity, PostgreSQL full-text search and a small entity/relationship graph, fused with reciprocal rank fusion and optionally reranked. The read path never calls a generative model.
- Isolates **tenants** with row-level security, and binds each credential to a tenant and optionally to a single subject.
- Exposes the same operations three ways: a Go package, a REST API, and an **MCP server** (stdio and Streamable HTTP).
- Ships an **evaluation harness** that reports accuracy on a public benchmark against three baselines, with latency p50/p95 and token cost per turn.

## What is different

| | mnemo (planned) | Mem0 | Graphiti (Zep) | Letta |
|---|---|---|---|---|
| Language | Go | Python, TypeScript | Python | Python |
| Licence | MIT | Apache-2.0 | Apache-2.0 | Apache-2.0 |
| Storage required | PostgreSQL + pgvector | not stated in README | Neo4j, FalkorDB or Neptune | not stated in README |
| Changed facts | superseded, validity interval kept | README describes ADD-only extraction | invalidated, validity window kept | not stated in README |
| Auth and tenant isolation in the open-source server | built in | not verified | reference servers ship without auth | not verified |

Sources: [mem0ai/mem0](https://github.com/mem0ai/mem0), [getzep/graphiti](https://github.com/getzep/graphiti), [letta-ai/letta](https://github.com/letta-ai/letta), READMEs read on 2026-11-19. "Not stated" means the README did not say, not that the feature is absent.

mnemo is not trying to beat these projects on features. The bet is narrower: a memory layer small enough to read in an afternoon, with one database, and a benchmark anyone can re-run with one command.

## Benchmark (planned)

Headline: accuracy on **LongMemEval_S** (500 questions, about 115k tokens of history per question, [repository](https://github.com/xiaowu0162/LongMemEval)), same reader model and same judge for every row.

| System | Accuracy | Knowledge-update | Temporal | Reader tokens / question | Recall p50 / p95 | Cost / turn |
|---|---|---|---|---|---|---|
| No memory | pending | pending | pending | pending | n/a | pending |
| Full context | pending | pending | pending | pending | n/a | pending |
| Plain vector top-k | pending | pending | pending | pending | pending | pending |
| **mnemo** | pending | pending | pending | pending | pending | pending |

Secondary: LoCoMo (10 conversations, categories 1-4), reported with its known defects. Ablations (lexical leg, facts, temporal validity, graph, reranker, consolidation, forgetting) are specified in [docs/evaluation-plan.md](docs/evaluation-plan.md).

Numbers published by other projects are listed in the evaluation plan with their sources. They were produced with different reader models, judges and protocols, so they are context, not rows of the table above.

## Planned architecture

One Go binary (`mnemo serve`) with an in-process worker, one PostgreSQL database. Writes are split in two: the episode is stored and acknowledged immediately; fact extraction runs as a job from a queue kept in the same database. Language models and embedding models are reached through OpenAI-compatible HTTP endpoints, so the service runs against a hosted provider or a local model without code changes.

```
agent / app ──REST / MCP / Go API──▶ mnemo ──SQL──▶ PostgreSQL (pgvector, full-text, pg_trgm)
                                       │
                                       └──HTTPS──▶ chat model · embedding model · reranker (optional)
```

Details, diagrams, data model and API: [docs/architecture.md](docs/architecture.md).

## Repository layout

```
README.md            this file
CONTRIBUTING.md      conventions, commands, Definition of Done
LICENSE              MIT
docs/
  blueprint.md       index of the design documents (start here)
  prd.md             problem, goals, metrics, requirements (FR-n)
  architecture.md    structure, data model, public API, diagrams
  adr/               architecture decision records
  security-baseline.md   threat model, controls, deletion
  quality-baseline.md    fitness functions, CI gates, reliability
  testing-strategy.md    test types per component, validation levels
  evaluation-plan.md     benchmarks, metrics, baselines, ablations
  open-questions.md      unresolved points and assumptions to confirm
  plan/              milestones and the ordered task list
```

Planned once implementation starts: `cmd/mnemo/` (binary), the root Go package (public API), `internal/` (store, extraction, resolution, retrieval, policy, providers, HTTP, MCP), `api/openapi.yaml`, `eval/` (Python harness), `docker-compose.yml`.

The documents under `docs/` are written in Spanish.

## Licence

MIT. See [LICENSE](LICENSE). Benchmark datasets are downloaded at run time and are not redistributed here; LoCoMo is CC BY-NC 4.0.
