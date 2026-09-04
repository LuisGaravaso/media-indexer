---
name: ticket-create
description: >-
  Creates a GitHub issue for a feature, bug, task, or specification (e.g. from docs/specs) and automatically adds it to GitHub Project #3 (@LuisGaravaso's Reeler). Triggered when the user invokes /ticket-create or asks to create/publish a ticket to GitHub.
---

# Ticket Create Skill (`/ticket-create`)

Creates a GitHub issue in `LuisGaravaso/media-manager` and links it directly to GitHub Project #3 (`@LuisGaravaso's Reeler`).

## Usage & Instructions

### 1. From a Specification File (e.g. `docs/specs/*.md`):
When given a spec path or commit number:
1. Extract title (from first `# ` header) and content.
2. Create the GitHub issue:
   ```bash
   issue_url=$(gh issue create --repo LuisGaravaso/media-manager --title "<Spec Title>" --body-file "<spec_file_path>")
   ```
3. Add the issue to GitHub Project #3:
   ```bash
   gh project item-add 3 --owner LuisGaravaso --url "$issue_url"
   ```
4. Output the issue number and URL to the user.

### 2. From Ad-hoc User Description:
1. Gather/formulate the title, acceptance criteria, and description.
2. Create the issue:
   ```bash
   issue_url=$(gh issue create --repo LuisGaravaso/media-manager --title "<Title>" --body "<Body>")
   ```
3. Add to Project #3:
   ```bash
   gh project item-add 3 --owner LuisGaravaso --url "$issue_url"
   ```
4. Confirm creation with issue URL.
