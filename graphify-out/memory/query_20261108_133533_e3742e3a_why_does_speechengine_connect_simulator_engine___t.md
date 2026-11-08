---
type: "query"
date: "2026-11-08T13:35:33.843278+00:00"
question: "Why does SpeechEngine connect Simulator Engine & Test Generation to 8 subsystems?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["SpeechEngine", "SimulatorEngine", "Server"]
---

# Q: Why does SpeechEngine connect Simulator Engine & Test Generation to 8 subsystems?

## Answer

SpeechEngine in pkg/engine/engine.go:L16 is the primary polymorphic interface implemented by both SimulatorEngine and SherpaRunner, consumed by Server, test harnesses, and evaluation suites.

## Outcome

- Signal: useful

## Source Nodes

- SpeechEngine
- SimulatorEngine
- Server