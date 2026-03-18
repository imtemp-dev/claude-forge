---
paths:
  - ".bts/**"
---

# BTS File Schema Reference

When creating or updating files in `.bts/state/`, you MUST follow these exact JSON schemas.
After creating or modifying any JSON file, run `bts validate` to verify compliance.

## manifest.json

```json
{
  "current_draft": "drafts/v3.md",
  "level": 2.5,
  "documents": {
    "research/v1.md": {
      "type": "research",
      "created_at": "2026-03-18T10:00:00Z",
      "based_on": [],
      "verified_by": ""
    },
    "drafts/v1.md": {
      "type": "draft",
      "created_at": "2026-03-18T10:30:00Z",
      "based_on": ["research/v1.md"],
      "incorporates": ["debates/001-auth-strategy"],
      "verified_by": "verifications/draft-v1.md"
    },
    "debates/001-auth-strategy": {
      "type": "debate",
      "created_at": "2026-03-18T11:00:00Z",
      "based_on": ["drafts/v1.md"]
    },
    "simulations/001-scenarios.md": {
      "type": "simulation",
      "created_at": "2026-03-18T12:00:00Z",
      "based_on": ["drafts/v2.md"],
      "resolves": []
    },
    "verifications/draft-v1.md": {
      "type": "verification",
      "created_at": "2026-03-18T10:35:00Z"
    }
  }
}
```

Required fields:
- `current_draft` (string): path to current draft version
- `level` (number): document level 0.0-3.0
- `documents` (object): keys are file paths, values are DocumentEntry objects

DocumentEntry required fields:
- `type` (string): one of "research", "draft", "debate", "simulation", "verification"
- `created_at` (string): ISO 8601 timestamp

DocumentEntry optional fields:
- `based_on` (array of strings): parent document paths
- `incorporates` (array of strings): debate/simulation paths incorporated
- `resolves` (array of strings): gap identifiers resolved
- `verified_by` (string): verification document path

## recipe.json

```json
{
  "id": "r-1710720000000",
  "type": "blueprint",
  "topic": "OAuth2 authentication",
  "phase": "verify",
  "iteration": 2,
  "draft_version": 3,
  "level": 2.5,
  "started_at": "2026-03-18T10:00:00Z",
  "updated_at": "2026-03-18T12:00:00Z"
}
```

Required fields:
- `id` (string): unique recipe identifier
- `type` (string): "analyze", "design", or "blueprint"
- `topic` (string): what the recipe is about
- `phase` (string): current phase — "research", "draft", "assess", "improve", "verify", "debate", "simulate", "audit", "finalize", "cancelled"
- `iteration` (number): current verify iteration count
- `draft_version` (number): current draft version number
- `level` (number): assessed document level 0.0-3.0
- `started_at` (string): ISO 8601 timestamp
- `updated_at` (string): ISO 8601 timestamp

## changelog.jsonl

Each line is a JSON object:

```json
{"time":"2026-03-18T10:00:00Z","action":"research","output":"research/v1.md"}
{"time":"2026-03-18T10:30:00Z","action":"draft","output":"drafts/v1.md","based_on":["research/v1.md"]}
{"time":"2026-03-18T10:35:00Z","action":"verify","input":"drafts/v1.md","result":"2 critical, 3 major"}
{"time":"2026-03-18T11:00:00Z","action":"improve","output":"drafts/v2.md","based_on":["drafts/v1.md"],"incorporates":["debates/001"]}
{"time":"2026-03-18T11:30:00Z","action":"debate","output":"debates/001-auth","result":"concluded: OAuth2"}
{"time":"2026-03-18T12:00:00Z","action":"simulate","output":"simulations/001.md","result":"4 gaps found"}
{"time":"2026-03-18T12:30:00Z","action":"assess","result":"Level 2.5","level":2.5}
```

Required fields:
- `time` (string): ISO 8601 timestamp. **Key name is "time", not "timestamp".**
- `action` (string): one of "research", "draft", "improve", "verify", "debate", "simulate", "audit", "assess", "sync-check", "finalize"

Optional fields:
- `input` (string): what was acted on
- `output` (string): what was produced
- `based_on` (array of strings): dependencies
- `incorporates` (array of strings): incorporated debates/simulations
- `resolves` (array of strings): resolved gaps
- `result` (string): summary of outcome
- `level` (number): level after this action

## debate meta.json

Located at `.bts/state/recipes/{id}/debates/{debate-id}/meta.json`:

```json
{
  "id": "001-auth-strategy",
  "topic": "OAuth2 vs JWT",
  "rounds": 3,
  "conclusion": "OAuth2 with Redis session cache",
  "decided": true,
  "started_at": "2026-03-18T11:00:00Z",
  "updated_at": "2026-03-18T11:30:00Z"
}
```

Required fields:
- `id` (string): debate identifier
- `topic` (string): debate topic
- `rounds` (number): number of completed rounds
- `decided` (boolean): whether a conclusion was reached
- `started_at` (string): ISO 8601 timestamp
- `updated_at` (string): ISO 8601 timestamp

Optional fields:
- `conclusion` (string): the decision reached (plain text, not object)

## IMPORTANT RULES

1. **Use exact field names** as shown above. `"time"` not `"timestamp"`. `"decided"` not `"status"`.
2. **`conclusion` is a string**, not an object. Write structured conclusions as a single sentence.
3. **`documents` in manifest is a flat map** where keys are file paths and values are DocumentEntry objects. Not categorized lists.
4. **Always run `bts validate` after creating/modifying any JSON file in `.bts/`.**
5. **Always create `recipe.json`** at the start of a recipe. This is how `bts recipe status` finds the active recipe.
