# Context

The 2023 Chatwork official user guide (page 18) shows a message link inserted by the message-link action in the exact shape `https://www.chatwork.com/#!rid<room-decimal>-<message-decimal>`; its adjacent room-ID help identifies the decimal immediately after `rid` as the room ID. The example is an observed documented form, not a complete URL grammar or proof of validity, current availability, or access. Sources: https://ec-commonassets.s3.amazonaws.com/site/chatwork/documents/pdf_ja_guide_v2.pdf and https://help.chatwork.com/hc/ja/articles/360000142942-ルームIDを確認する . Only synthetic references will be used in code and tests.

Feedback item 6 says the typical starting point is a user-pasted message link, and requires discovery and action to remain separate. The existing canonical positive decimal reference rule allows 1..32 digits without leading zero. Command selection is an invocation boundary and the new leaf must be configurable. Neither the source URL nor an output reference proves access or existence.


## Exploratory agent run, 2026-11-10

A fresh `gpt-6-luna` max agent was given only a synthetic link, the task of extracting its room/message refs, and the path to a production `cwk` binary built from a source copy. The copy's 522 listed tracked/unignored entries matched their original content hashes after the run; this is not an atomic snapshot proof. The CLI process had a fresh temporary XDG config enabling only `messages resolve-link`, and an empty inherited environment except a fixed PATH and that XDG path. No Chatwork credential was supplied to the CLI.

The wrapper recorded three calls: root help, resolver help, then `messages resolve-link --url` with the synthetic link `https://www.chatwork.com/#!rid42-101`. The agent reported `room-ref=42 message-ref=101 existence=not-checked access=not-checked`, matching a separate pre-run smoke check. The wrapper records space-joined argument strings, not a lossless argv array, output, or all agent tools; the answer and successful command sequence are agent-reported and the CLI output was not independently captured during the run. Agent tool/network isolation and usage counts were not independently verified. This single exploratory observation does not establish real message access, representative success rates, or a settings/no-settings comparison.
