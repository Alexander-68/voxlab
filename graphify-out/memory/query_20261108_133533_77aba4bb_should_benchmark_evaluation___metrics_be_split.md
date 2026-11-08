---
type: "query"
date: "2026-11-08T13:35:33.576322+00:00"
question: "Should Benchmark Evaluation & Metrics be split?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["BaseUtterance", "EvalRunner", "LatencyStatistics"]
---

# Q: Should Benchmark Evaluation & Metrics be split?

## Answer

Yes. Cohesion is 0.069 because it groups benchmark metric computation (pkg/evaluation/metrics.go), test harness execution (runner.go), and synthetic audio corpus perturbation (generator.go, corpus.go).

## Outcome

- Signal: useful

## Source Nodes

- BaseUtterance
- EvalRunner
- LatencyStatistics