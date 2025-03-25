# Aggregate expression engines

This repo contains Inngest's aggregate expression engine service, turning O(n^2) expression
matching into O(n).

It does this by:

1. Parsing each expression whilst lifting literals out of expressions
2. Breaking expressions down into subgroups (matching && comparators)
3. Storing each group's comparator in a matching engine for fast lookups

When an event is received, instead of matching the event against every expression, we instead
inspect each matching engine to filter out invalid expressions.  This leaves us with a subset of
expressions that are almost always matching for the event, simplifying the number of expressions
to match against.

Copyright Inngest 2024.
