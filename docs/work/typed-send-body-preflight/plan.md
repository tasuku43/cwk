# Work Plan: Typed send body-file preflight

- Status: Implemented, awaiting review
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Read the body file through the existing infrastructure-owned bounded reader before the reply GET, validate raw bytes, and return a one-use opaque handle. Keep the handle inside the adapter; application code only carries it between the body preflight, exact reply read, and final send. On all pre-dispatch exits, discard and clear the handle. On send, consume and clear it before entering the existing single-attempt transport path.

## Alternatives considered

### Move filesystem access into the application

Rejected because it widens application capability and violates the controlled side-effect boundary.

### Read the body file again when posting

Rejected because it creates a TOCTOU window and could send bytes different from those validated before the reply GET.

### Weaken the zero-provider-call contract

Rejected because the security contract already promises this behavior for invalid body sources.

## Verification

- Focused application, infrastructure, CLI, and domain tests.
- Integration call-count, exact form/body suffix, same-binding/context, handle cleanup/redaction, pre-dispatch failure, and post-dispatch unknown-outcome evidence.
- Root agent runs the repository gate after shared work settles.
