// Package policy is the policy domain module of EBC-X.
//
// Module boundary: this package owns its aggregate roots and domain events.
// Cross-module communication is via Domain Event + Orchestrator, never direct calls (TASK-R03).
package policy
