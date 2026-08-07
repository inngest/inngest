import assert from 'node:assert';
import { describe, it } from 'vitest';

import { looksLikeAIOutput } from './utils';

const googleOutput = {
  body: {
    response: {
      candidates: [
        {
          content: {
            parts: [
              {
                text: 'A function calls self,\nSolving problems by layers,\nA fractal of code. \n',
              },
            ],
            role: 'model',
          },
          finishReason: 'STOP',
          index: 0,
        },
      ],
      modelVersion: 'gemini-1.5-flash-001',
      usageMetadata: {
        candidatesTokenCount: 17,
        promptTokenCount: 9,
        totalTokenCount: 26,
      },
    },
  },
  event: {
    data: {
      model: 'gemini-1.5-flash',
      prompt: 'Write a haiku about recursion in programming.',
      provider: 'google',
    },
    id: '01JC6AHCZAGTDDAXN95H4XW8DP',
    name: 'inngest/function.invoked',
    ts: 1731084202986,
    user: {},
  },
};

const anthropicOutput = {
  id: 'msg_01Eu6wq1Dt6FFPMsxhpnPVfM',
  type: 'message',
  role: 'assistant',
  model: 'claude-3-5-sonnet-20241022',
  content: [
    {
      type: 'text',
      text: "Here's a haiku about recursion:\n\nFunction calls itself\nUntil base case is reached, then\nReturns back through time",
    },
  ],
  stop_reason: 'end_turn',
  stop_sequence: null,
  usage: {
    input_tokens: 17,
    output_tokens: 35,
  },
};

const openAIOutput = {
  id: 'chatcmpl-AQd7Vqr5yNdAeoQC5yra9XXsaTRth',
  object: 'chat.completion',
  created: 1730910269,
  model: 'gpt-4o-mini-2024-07-18',
  choices: [
    {
      index: 0,
      message: {
        role: 'assistant',
        content:
          'Functions call themselves,  \nLayers of thought intertwine—  \nEndless loops of code.',
        refusal: null,
      },
      logprobs: null,
      finish_reason: 'stop',
    },
  ],
  usage: {
    prompt_tokens: 16,
    completion_tokens: 18,
    total_tokens: 34,
  },
  system_fingerprint: 'fp_0ba0d124f1',
};

const vercelOutput = {
  body: {
    experimental_providerMetadata: {
      openai: {
        cachedPromptTokens: 0,
        reasoningTokens: 0,
      },
    },
    finishReason: 'stop',
    text: "Functions call themselves,  \nLayers deep in logic's dance,  \nEndless loops of thought.",
    usage: {
      completionTokens: 19,
      promptTokens: 16,
      totalTokens: 35,
    },
  },
  event: {
    data: {
      model: 'gpt-4o-mini',
      prompt: 'Write a haiku about recursion in programming.',
      provider: 'vercel',
    },
    id: '01JBY9WZKXKKPNGAPFD5GASBQX',
    name: 'inngest/function.invoked',
    ts: 1730815098493,
    user: {},
  },
};

describe('looksLikeAIOutput', () => {
  it('returns true for open ai output', () => {
    assert.strictEqual(looksLikeAIOutput(JSON.stringify(openAIOutput)), true);
  });

  it('returns true for anthropic output', () => {
    assert.strictEqual(looksLikeAIOutput(JSON.stringify(anthropicOutput)), true);
  });

  it('returns true for vercel ai sdk output nested under body', () => {
    assert.strictEqual(looksLikeAIOutput(JSON.stringify(vercelOutput)), true);
  });

  it('returns true for google output nested under body', () => {
    assert.strictEqual(looksLikeAIOutput(JSON.stringify(googleOutput)), true);
  });

  it('returns true for step output nested under data', () => {
    assert.strictEqual(looksLikeAIOutput(JSON.stringify({ data: openAIOutput })), true);
  });

  it('returns false for ordinary JSON output', () => {
    assert.strictEqual(looksLikeAIOutput(JSON.stringify({ ok: true, count: 3 })), false);
  });

  it('returns false for JSON with incidental key matches', () => {
    assert.strictEqual(
      looksLikeAIOutput(JSON.stringify({ output: 'done', total: 42, promptTokens: 5 })),
      false
    );
  });

  it('returns false for non-JSON output', () => {
    assert.strictEqual(looksLikeAIOutput('not json'), false);
  });

  it('returns false for JSON primitives', () => {
    assert.strictEqual(looksLikeAIOutput('"hello"'), false);
    assert.strictEqual(looksLikeAIOutput('42'), false);
    assert.strictEqual(looksLikeAIOutput('null'), false);
  });
});
