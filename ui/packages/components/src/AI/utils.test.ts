import assert from 'node:assert';
import { describe, it } from 'vitest';

import { getAIInfo, type OpenAIOutput } from './utils';

const openAIOutput: OpenAIOutput = {
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
    prompt_tokens_details: {
      cached_tokens: 0,
      audio_tokens: 0,
    },
    completion_tokens_details: {
      reasoning_tokens: 0,
      audio_tokens: 0,
      accepted_prediction_tokens: 0,
      rejected_prediction_tokens: 0,
    },
  },

  system_fingerprint: 'fp_0ba0d124f1',
};

const vercelOutput: VercelOutput = {
  body: {
    experimental_providerMetadata: {
      openai: {
        cachedPromptTokens: 0,
        reasoningTokens: 0,
      },
    },
    finishReason: 'stop',
    rawResponse: {
      headers: {
        'access-control-expose-headers': 'X-Request-ID',
        'alt-svc': 'h3=":443"; ma=86400',
        'cf-cache-status': 'DYNAMIC',
        'cf-ray': '8ddd5b9ea9116ac7-BOS',
        'content-type': 'application/json',
        date: 'Tue, 05 Nov 2024 13:58:19 GMT',
        'openai-organization': 'user-boy2kh9lgjptqilzbxyxni1k',
        'openai-processing-ms': '452',
        'openai-version': '2020-10-01',
        server: 'cloudflare',
        'set-cookie':
          '_cfuvid=MmlDGCvdN5pZW9QmOm9Cy56lYtQQc6Y0B1YJO5zeCyg-1730815099270-0.0.1.1-604800000; path=/; domain=.api.openai.com; HttpOnly; Secure; SameSite=None',
        'strict-transport-security': 'max-age=31536000; includeSubDomains; preload',
        'x-content-type-options': 'nosniff',
        'x-ratelimit-limit-requests': '10000',
        'x-ratelimit-limit-tokens': '200000',
        'x-ratelimit-remaining-requests': '9999',
        'x-ratelimit-remaining-tokens': '199971',
        'x-ratelimit-reset-requests': '8.64s',
        'x-ratelimit-reset-tokens': '8ms',
        'x-request-id': 'req_9578dfea05f01c1232d458f84aeecba6',
      },
    },
    request: {
      body: '{"model":"gpt-4o-mini","temperature":0,"messages":[{"role":"user","content":"Write a haiku about recursion in programming."}]}',
    },
    response: {
      headers: {
        'access-control-expose-headers': 'X-Request-ID',
        'alt-svc': 'h3=":443"; ma=86400',
        'cf-cache-status': 'DYNAMIC',
        'cf-ray': '8ddd5b9ea9116ac7-BOS',
        'content-type': 'application/json',
        date: 'Tue, 05 Nov 2024 13:58:19 GMT',
        'openai-organization': 'user-boy2kh9lgjptqilzbxyxni1k',
        'openai-processing-ms': '452',
        'openai-version': '2020-10-01',
        server: 'cloudflare',
        'set-cookie':
          '_cfuvid=MmlDGCvdN5pZW9QmOm9Cy56lYtQQc6Y0B1YJO5zeCyg-1730815099270-0.0.1.1-604800000; path=/; domain=.api.openai.com; HttpOnly; Secure; SameSite=None',
        'strict-transport-security': 'max-age=31536000; includeSubDomains; preload',
        'x-content-type-options': 'nosniff',
        'x-ratelimit-limit-requests': '10000',
        'x-ratelimit-limit-tokens': '200000',
        'x-ratelimit-remaining-requests': '9999',
        'x-ratelimit-remaining-tokens': '199971',
        'x-ratelimit-reset-requests': '8.64s',
        'x-ratelimit-reset-tokens': '8ms',
        'x-request-id': 'req_9578dfea05f01c1232d458f84aeecba6',
      },
      id: 'chatcmpl-AQEMUid08GE2zgLbROot3QuxVGaCf',
      messages: [
        {
          content: [
            {
              text: "Functions call themselves,  \nLayers deep in logic's dance,  \nEndless loops of thought.",
              type: 'text',
            },
          ],
          role: 'assistant',
        },
      ],
      modelId: 'gpt-4o-mini-2024-07-18',
      timestamp: '2024-11-05T13:58:18.000Z',
    },
    responseMessages: [
      {
        content: [
          {
            text: "Functions call themselves,  \nLayers deep in logic's dance,  \nEndless loops of thought.",
            type: 'text',
          },
        ],
        role: 'assistant',
      },
    ],
    roundtrips: [
      {
        experimental_providerMetadata: {
          openai: {
            cachedPromptTokens: 0,
            reasoningTokens: 0,
          },
        },
        finishReason: 'stop',
        isContinued: false,
        request: {
          body: '{"model":"gpt-4o-mini","temperature":0,"messages":[{"role":"user","content":"Write a haiku about recursion in programming."}]}',
        },
        response: {
          headers: {
            'access-control-expose-headers': 'X-Request-ID',
            'alt-svc': 'h3=":443"; ma=86400',
            'cf-cache-status': 'DYNAMIC',
            'cf-ray': '8ddd5b9ea9116ac7-BOS',
            'content-type': 'application/json',
            date: 'Tue, 05 Nov 2024 13:58:19 GMT',
            'openai-organization': 'user-boy2kh9lgjptqilzbxyxni1k',
            'openai-processing-ms': '452',
            'openai-version': '2020-10-01',
            server: 'cloudflare',
            'set-cookie':
              '_cfuvid=MmlDGCvdN5pZW9QmOm9Cy56lYtQQc6Y0B1YJO5zeCyg-1730815099270-0.0.1.1-604800000; path=/; domain=.api.openai.com; HttpOnly; Secure; SameSite=None',
            'strict-transport-security': 'max-age=31536000; includeSubDomains; preload',
            'x-content-type-options': 'nosniff',
            'x-ratelimit-limit-requests': '10000',
            'x-ratelimit-limit-tokens': '200000',
            'x-ratelimit-remaining-requests': '9999',
            'x-ratelimit-remaining-tokens': '199971',
            'x-ratelimit-reset-requests': '8.64s',
            'x-ratelimit-reset-tokens': '8ms',
            'x-request-id': 'req_9578dfea05f01c1232d458f84aeecba6',
          },
          id: 'chatcmpl-AQEMUid08GE2zgLbROot3QuxVGaCf',
          messages: [
            {
              content: [
                {
                  text: "Functions call themselves,  \nLayers deep in logic's dance,  \nEndless loops of thought.",
                  type: 'text',
                },
              ],
              role: 'assistant',
            },
          ],
          modelId: 'gpt-4o-mini-2024-07-18',
          timestamp: '2024-11-05T13:58:18.000Z',
        },
        stepType: 'initial',
        text: "Functions call themselves,  \nLayers deep in logic's dance,  \nEndless loops of thought.",
        toolCalls: [],
        toolResults: [],
        usage: {
          completionTokens: 19,
          promptTokens: 16,
          totalTokens: 35,
        },
        warnings: [],
      },
    ],
    steps: [
      {
        experimental_providerMetadata: {
          openai: {
            cachedPromptTokens: 0,
            reasoningTokens: 0,
          },
        },
        finishReason: 'stop',
        isContinued: false,
        request: {
          body: '{"model":"gpt-4o-mini","temperature":0,"messages":[{"role":"user","content":"Write a haiku about recursion in programming."}]}',
        },
        response: {
          headers: {
            'access-control-expose-headers': 'X-Request-ID',
            'alt-svc': 'h3=":443"; ma=86400',
            'cf-cache-status': 'DYNAMIC',
            'cf-ray': '8ddd5b9ea9116ac7-BOS',
            'content-type': 'application/json',
            date: 'Tue, 05 Nov 2024 13:58:19 GMT',
            'openai-organization': 'user-boy2kh9lgjptqilzbxyxni1k',
            'openai-processing-ms': '452',
            'openai-version': '2020-10-01',
            server: 'cloudflare',
            'set-cookie':
              '_cfuvid=MmlDGCvdN5pZW9QmOm9Cy56lYtQQc6Y0B1YJO5zeCyg-1730815099270-0.0.1.1-604800000; path=/; domain=.api.openai.com; HttpOnly; Secure; SameSite=None',
            'strict-transport-security': 'max-age=31536000; includeSubDomains; preload',
            'x-content-type-options': 'nosniff',
            'x-ratelimit-limit-requests': '10000',
            'x-ratelimit-limit-tokens': '200000',
            'x-ratelimit-remaining-requests': '9999',
            'x-ratelimit-remaining-tokens': '199971',
            'x-ratelimit-reset-requests': '8.64s',
            'x-ratelimit-reset-tokens': '8ms',
            'x-request-id': 'req_9578dfea05f01c1232d458f84aeecba6',
          },
          id: 'chatcmpl-AQEMUid08GE2zgLbROot3QuxVGaCf',
          messages: [
            {
              content: [
                {
                  text: "Functions call themselves,  \nLayers deep in logic's dance,  \nEndless loops of thought.",
                  type: 'text',
                },
              ],
              role: 'assistant',
            },
          ],
          modelId: 'gpt-4o-mini-2024-07-18',
          timestamp: '2024-11-05T13:58:18.000Z',
        },
        stepType: 'initial',
        text: "Functions call themselves,  \nLayers deep in logic's dance,  \nEndless loops of thought.",
        toolCalls: [],
        toolResults: [],
        usage: {
          completionTokens: 19,
          promptTokens: 16,
          totalTokens: 35,
        },
        warnings: [],
      },
    ],
    text: "Functions call themselves,  \nLayers deep in logic's dance,  \nEndless loops of thought.",
    toolCalls: [],
    toolResults: [],
    usage: {
      completionTokens: 19,
      promptTokens: 16,
      totalTokens: 35,
    },
    warnings: [],
  },
  event: {
    data: {
      _inngest: {
        expire: 0,
        fn_id: 'inngest-ai-generate-text',
        gid: '',
        name: '',
        source_app_id: '',
        source_fn_id: '',
        source_fn_v: 0,
      },
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

describe('parseAIOutput', (t) => {
  it('test open ai output parsing', () => {
    const aiInfo = getAIInfo(openAIOutput);

    assert.deepStrictEqual(aiInfo, {
      model: 'gpt-4o-mini-2024-07-18',
      promptTokens: 16,
      completionTokens: 18,
      totalTokens: 34,
    });
  });

  it('test vercel ai sdk output parsing', () => {
    const aiInfo = getAIInfo(vercelOutput);

    assert.deepStrictEqual(aiInfo, {
      completionTokens: 19,
      promptTokens: 16,
      totalTokens: 35,
      model: 'gpt-4o-mini-2024-07-18',
    });
  });
});
