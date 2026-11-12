# Context

Feedback item 7 identifies repeated To, reply and quote notation in message bodies and requests a separate semantic body and an evidence-based token comparison. The existing infrastructure parser recognizes reviewed To, reply, quote and code forms; it currently returns typed relation metadata while retaining the exact body. A quote relation records author/time, not the quoted text or its position, so simply removing an entire quote would lose meaning. Tags inside quote and code are data. Malformed recognized syntax makes the complete relation set unknown.

An official documented grammar for system-notification `dtext` and exact object linkage has not been established. Notification classification remains separate and unresolved; do not infer an object or kind from natural language or unverified syntax. The current headerless public projection is unchanged until a comparison meets the project protocol.

The candidate now retains source offsets and explicit removed spans; the domain checks source coverage while the adapter owns provider-notation classification. Independent reviews identified missing source coverage and retained body references after send reconciliation; both were fixed and retested. A separate fresh review approved the added coverage tests. Focused race tests and `task check` passed. No public presentation comparison or scored agent run has occurred.


## Notification evidence, 2026-11-12

The user cannot provide anonymized notification examples. The official [message notation documentation](https://developer.chatwork.com/docs/message-notation) describes `[info]` as a general message-formatting tag, which can be used to present system-like text; its presence therefore does not establish a provider-generated notification. The consulted official message, file and task API references do not supply an authoritative event-kind grammar for `dtext` or prove that a visible notification text identifies its referenced object. A general web search is not proof such a grammar cannot exist. Do not classify provider events by text, punctuation, `[info]`, or a guessed `dtext` format.

The already-implemented `messages resources` is a partial, separate answer to the file/task workflow: it joins the typed `message_id` from bounded file and room-task collections to the selected room/message and reports incomplete coverage and unknown absence. It does not establish that the message is a system notification or that a task was created, completed, or edited by a particular event. It is therefore not a substitute for notification typing. Notification event classification and its authoritative evidence remain unresolved; the user has not been asked to perform a further action.
