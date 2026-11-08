---
type: "query"
date: "2026-11-08T13:35:32.191024+00:00"
question: "Why does Server connect HTTP Web Server & Networking to 14 other subsystems?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Server", "NewServer()", "DSPProcessor", "StateMachine"]
---

# Q: Why does Server connect HTTP Web Server & Networking to 14 other subsystems?

## Answer

Server in pkg/server/server.go:L35 acts as the central stateful orchestrator holding audio sources, DSP preprocessors, speech enhancers, ring buffers, state machines, TTS managers, intent matchers, websocket clients, and evaluation runners.

## Outcome

- Signal: useful

## Source Nodes

- Server
- NewServer()
- DSPProcessor
- StateMachine