type AgenticGuideOptions = {
  apiBaseUrl: string;
  serverOrigin: string;
};

export const buildAgenticGuides = ({
  apiBaseUrl,
  serverOrigin,
}: AgenticGuideOptions) => {
  const cliBase = `${serverOrigin}`;

  const runCommand = `curl "${apiBaseUrl}/runs/<run_id>?include_output=true"`;
  const traceCommand = `curl "${apiBaseUrl}/runs/<run_id>/trace?include_output=true"`;
  const invokeCommand = `curl -X POST "${apiBaseUrl}/functions/<function_slug>/invoke" \\
  -H "Content-Type: application/json" \\
  -d '{
    "data": {
      "message": "hello from local agent test"
    }
  }'`;

  const cliExamples = `inngest runs get <run_id> --api-url ${cliBase}
inngest runs trace <run_id> --api-url ${cliBase} --include-output
inngest functions invoke <function_slug> --api-url ${cliBase} --data '{"message":"hello"}'`;

  const skillsMd = `# Inngest Dev Server Agent Skills

Use the local Inngest dev server first before reaching for the dashboard.

## Local API

- Base URL: ${apiBaseUrl}
- Auth: none required in local dev
- If a client insists on a token, any \`sk-inn-api-*\` value is acceptable for local testing

## Primary commands

\`\`\`bash
${cliExamples}
\`\`\`

## HTTP endpoints

\`\`\`bash
${runCommand}

${traceCommand}

${invokeCommand}
\`\`\`

## Suggested debugging loop

1. Invoke the function by slug.
2. Capture the returned run ID.
3. Fetch the run summary with output enabled.
4. Fetch the trace with output enabled.
5. If a step fails, inspect the failing span's input, output, and response metadata.
`;

  const claudeMd = `# Inngest Local Agent Instructions

When working against this repo's dev server, prefer the local Inngest API and CLI instead of the cloud dashboard.

## Use this server

- Dev server root: ${serverOrigin}
- Agentic API base: ${apiBaseUrl}
- MCP endpoint: ${serverOrigin}/mcp

## What to do first

- Confirm the function slug exists locally.
- Invoke by slug to produce a fresh run.
- Use the run and trace endpoints to validate behavior end to end.

## Useful examples

\`\`\`bash
${invokeCommand}

${runCommand}

${traceCommand}
\`\`\`

## Local auth notes

- No API key is required in dev mode.
- If a tool requires one, use a placeholder \`sk-inn-api-local\`.
`;

  return {
    apiBaseUrl,
    cliBase,
    runCommand,
    traceCommand,
    invokeCommand,
    cliExamples,
    skillsMd,
    claudeMd,
  };
};
