#!/usr/bin/env node
/**
 * Capture a fixture from a running Dev Server.
 *
 *   node capture.mjs <runID> <name>
 *
 * Issues the same `GetRun` query the UI issues — kept in step with
 * `ui/apps/dev-server-ui/src/coreapi.ts` — and writes `{ run: { status, trace } }`
 * to `<name>.json`, which is the shape every other fixture is in.
 *
 * The README documented this as "issue the GetRun query and save the data
 * object", which is accurate and takes fifteen minutes to do by hand. This is
 * the same thing, repeatable.
 */
import { writeFileSync } from 'node:fs';
import { resolve } from 'node:path';

const TRACE_DETAILS = `
  fragment TraceDetails on RunTraceSpan {
    name
    status
    attempts
    queuedAt
    scheduledAt
    startedAt
    endedAt
    isRoot
    isUserland
    userlandSpan {
      spanName
      spanKind
      serviceName
      scopeName
      scopeVersion
      spanAttrs
      resourceAttrs
    }
    metadata {
      scope
      kind
      values
      updatedAt
    }
    outputID
    groupID
    plannedSteps {
      stepID
      name
      stepOp
    }
    userlandStepID
    userlandStepIndex
    parentStepIDs
    parentAlternateStepIDs
    debugRunID
    debugSessionID
    spanID
    stepID
    stepOp
    stepType
    stepInfo {
      __typename
      ... on InvokeStepInfo {
        triggeringEventID
        functionID
        timeout
        returnEventID
        runID
        timedOut
      }
      ... on SleepStepInfo {
        sleepUntil
      }
      ... on WaitForEventStepInfo {
        eventName
        expression
        timeout
        foundEventID
        timedOut
      }
      ... on RunStepInfo {
        type
      }
      ... on WaitForSignalStepInfo {
        signal
        timeout
        timedOut
      }
    }
    response {
      statusCode
      headers
    }
  }
`;

const GET_RUN = `
  query GetRun($runID: String!) {
    run(runID: $runID) {
      status
      trace {
        discoveries {
          spanID
          status
          queuedAt
          startedAt
          endedAt
          plannedStepIDs
        }
        ...TraceDetails
        childrenSpans {
          ...TraceDetails
          childrenSpans {
            ...TraceDetails
            childrenSpans {
              ...TraceDetails
              childrenSpans {
                ...TraceDetails
              }
            }
          }
        }
      }
    }
  }
  ${TRACE_DETAILS}
`;

const [runID, name] = process.argv.slice(2);
if (!runID || !name) {
  console.error('usage: node capture.mjs <runID> <name>');
  process.exit(1);
}

const endpoint = process.env.INNGEST_GQL ?? 'http://127.0.0.1:8288/v0/gql';

const res = await fetch(endpoint, {
  method: 'POST',
  headers: { 'content-type': 'application/json' },
  body: JSON.stringify({ query: GET_RUN, variables: { runID } }),
});

const body = await res.json();
if (body.errors) {
  console.error(JSON.stringify(body.errors, null, 2));
  process.exit(1);
}
if (!body.data?.run) {
  console.error(`no run ${runID}`);
  process.exit(1);
}

const out = resolve(import.meta.dirname, `${name}.json`);
writeFileSync(out, `${JSON.stringify({ run: body.data.run }, null, 2)}\n`);

let spans = 0;
const walk = (t) => {
  spans += 1;
  (t.childrenSpans ?? []).forEach(walk);
};
walk(body.data.run.trace);

console.log(`${name}.json  status=${body.data.run.status}  spans=${spans}`);
