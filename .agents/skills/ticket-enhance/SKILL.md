---
name: ticket-enhance
description: >-
  Analyzes an existing GitHub issue against the current state of the codebase, detects obsolete assumptions, missing requirements (e.g. Makefile targets, project structure changes, architectural decisions), and proposes or updates the issue description and acceptance criteria on GitHub. Triggered when the user invokes /ticket-enhance or asks to review/enhance a ticket before starting.
---

# Ticket Enhance Skill (`/ticket-enhance`)

Reviews a GitHub issue against the current codebase state, identifies drifts or missing details, and enhances the issue description and acceptance criteria directly on GitHub before development starts.

## Procedure

1. **Fetch the Ticket:**
   - Fetch the issue by number:
     ```bash
     gh issue view <issue-number> --repo LuisGaravaso/media-manager --json number,title,body,state,url,labels
     ```

2. **Audit Current Codebase Context:**
   - Inspect existing project structure, configuration files, and tooling:
     - `Makefile` (check if new targets should be specified in the ticket).
     - `go.mod` / dependencies (strictly Go 1.26, modern libraries).
     - Architecture patterns established in previous commits.
     - `AGENTS.md` rules and guidelines.

3. **Identify Enhancements & Delta:**
   - Check if ticket assumptions are outdated (e.g. references to Go 1.24 or other versions, references to obsolete `docs/` files, missing Makefile targets, outdated test commands).
   - Ensure all references to Go toolchain and Docker base images strictly use Go 1.26 (`golang:1.26-alpine`).
   - Add explicit acceptance criteria, edge cases, and file lists matching current architecture.
   - Formulate clear, structured markdown improvements to the issue body.

4. **Update or Propose Updates:**
   - Present the identified gaps/enhancements to the user.
   - With confirmation, update the GitHub issue directly:
     ```bash
     gh issue edit <issue-number> --repo LuisGaravaso/media-manager --body "<updated-body>"
     ```
   - Add relevant labels if applicable (e.g. `ready`, `enhanced`).

5. **Next Steps:**
   - Inform the user that the ticket is updated and ready to be started with `/ticket-start <issue-number>`.
