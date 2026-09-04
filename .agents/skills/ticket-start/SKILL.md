---
name: ticket-start
description: >-
  Fetches a GitHub issue / project ticket by issue number, URL, or title from LuisGaravaso/media-manager, reads its requirements and acceptance criteria, moves/assigns status if appropriate, and begins step-by-step implementation. Triggered when the user invokes /ticket-start or asks to start working on a ticket.
---

# Ticket Start Skill (`/ticket-start`)

Fetches details for a given GitHub issue / ticket, prepares the development context, and guides or starts the implementation following repository guidelines.

## Procedure

1. **Identify the Ticket:**
   - If user provided an issue number (e.g. `#1` or `1`):
     ```bash
     gh issue view 1 --repo LuisGaravaso/media-manager --json number,title,body,state,url,assignees,labels
     ```
   - If user provided a query/title or asked for the next pending ticket:
     ```bash
     gh issue list --repo LuisGaravaso/media-manager --state open --limit 10
     ```

2. **Understand Requirements & Context:**
   - Review issue description, acceptance criteria, and relevant files mentioned in the ticket.
   - Cross-reference with `AGENTS.md` and repository conventions (e.g., Testify testing standards, atomic commits, branch conventions).

3. **Branch Creation & Isolation (MANDATORY):**
   - All code modifications MUST occur in a dedicated feature branch. Never commit or edit code directly on `main`.
   - Ensure `main` is up to date:
     ```bash
     git checkout main && git pull origin main
     ```
   - Create and switch to a new feature branch for the ticket:
     ```bash
     git checkout -b feat/issue-<number>-<short-slug>
     ```
   - Verify active branch before writing code:
     ```bash
     git branch --show-current
     ```

4. **Implementation & Testing:**
   - Make the required changes incrementally.
   - Run tests (`go test -v -race ./...`) and linting (`go vet ./...` / `golangci-lint run`).
   - Ensure all acceptance criteria from the ticket are verified.

5. **Report & Next Steps:**
   - Summarize implemented changes and test results.
   - Propose commit / PR when ready.
