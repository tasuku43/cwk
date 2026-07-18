# Context: Positional collection records

## Verified facts

- The repository profile is `ready` and the working tree is clean.
- Collection headers already own count, positive limit, and completeness once.
- All seven result slices preserve provider order and use fixed typed item
  structures. Presentation does not need semantic or adapter changes.
- Current item lines repeat canonical field labels and `untrusted:` for every
  external-text value.
- `files list` always emits a message reference position; the typed zero value
  renders as the explicit atom `absent`.
- Contact organization is an optional labeled suffix because its name and
  department subfields are independently optional. Required leading positions
  never shift.
- Contact-request message is an optional final quoted position; omission cannot
  shift another field.
- Existing terminal-safe `quoted` and `atom` helpers are reusable unchanged.
- The active six-file fixture measures 218 tokens for the preceding labeled
  output and 146 for the fixed positional output under `tiktoken==0.13.0`
  `o200k_base`, while preserving the same typed semantic answer.

## Fixed schemas

```text
contacts: account-ref room-ref "name" [organization]
rooms: room-ref "name" type role unread mentions tasks
members: account-ref "name" role
personal-tasks: task-ref room-ref assigned-by-ref message-ref "body" status
room-tasks: task-ref room-ref account-ref message-ref "body" status limit-time
files: file-ref room-ref account-ref message-ref "name" size
contact-requests: request-ref account-ref "name" ["message"]
```

Canonical reference values remain literal fields. Text is quoted under one
collection-level trust declaration. Optional suffixes do not alter the required
field positions.
