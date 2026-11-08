---
type: "query"
date: "2026-11-08T13:35:32.459124+00:00"
question: "Are the 16 inferred relationships involving setupTestServer() correct?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["setupTestServer()", "TestDefaultBindIsLoopback()", "TestDictationIsIsolatedFromOtherActions()"]
---

# Q: Are the 16 inferred relationships involving setupTestServer() correct?

## Answer

Verified correct: setupTestServer() in pkg/server/server_test.go:L23 is invoked directly by 16 test functions across lifecycle_test.go (6), security_test.go (8), turn_test.go (2), and server_test.go (12) as the shared test fixture.

## Outcome

- Signal: useful

## Source Nodes

- setupTestServer()
- TestDefaultBindIsLoopback()
- TestDictationIsIsolatedFromOtherActions()