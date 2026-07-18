# Context

> Superseded implementation decision (2026-07-18): the axioms remain
> representation-independent, while
> `docs/work/chatwork-public-api-v2/goal.md` selects candidate C as the first
> stable implementation contract and defers comparative optimization.

## Verified product evidence

- The primary product concern is reducing an agent's cognitive load and token consumption while preserving task correctness.
- An agent should not be uncertain about which command satisfies a user's request or how to invoke it.
- A supported task should not require routine `jq`, `grep`, account/name joins, raw Chatwork-notation parsing, or exploratory API calls.
- Chatwork data contains relationships and repeated values that raw provider JSON may express inefficiently.
- Several concrete encodings could improve the result, but no encoding has yet been measured against representative agent tasks.
- The project owner intends to compare presentation implementations in parallel worktrees before selecting a public contract.

## Correction to earlier exploration

Dictionary compression, local aliases, indentation, Context Capsules, normalized JSON, and task-specific compact grammars were examples used to expose product needs. None is currently an axiom or selected public format.

The durable conclusions are instead:

- semantics required for the task must be available and truthful;
- missing context and uncertainty must be observable;
- identity and side effects must remain exact;
- transformation owned by a supported task should not be shifted to the agent;
- presentation should minimize token cost while maintaining or improving measured understanding quality;
- the winner must be chosen from evidence produced under comparable conditions.

## Unknowns

- Presentation candidates and how many worktrees participate in the first competition.
- Fixture corpus and target Chatwork scenarios.
- Target agent/model set, repetitions, prompts, and scoring method.
- Pinned tokenizer or token accounting source.
- Numerical quality floor and token budget.
- Whether one format can serve humans and agents or separate formats are justified.
