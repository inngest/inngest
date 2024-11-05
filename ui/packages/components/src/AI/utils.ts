export type OpenAIOutput = {
  model: string;
  object: string;
  usage: {
    completion_tokens: number;
    completion_tokens_details: {
      accepted_prediction_tokens: number;
      reasoning_tokens: number;
      rejected_prediction_tokens: number;
    };
    prompt_tokens: number;
    prompt_tokens_details: {
      cached_tokens: number;
    };
    total_tokens: number;
  };
  experimental_providerMetadata?: never;
};
export type VercelAIOutput = {
  experimental_providerMetadata: unknown;
  roundtrips: [{ response: { modelId: string } }];
  usage: {
    completionTokens: number;
    promptTokens: number;
    totalTokens: number;
  };
  model?: never;
};

export type ExperimentalAI = OpenAIOutput | VercelAIOutput;

export const parseAIOutput = (output: string): ExperimentalAI | undefined => {
  try {
    const data: ExperimentalAI = JSON.parse(output);

    //
    // a temporary hack to detect ai output until first class
    // step.ai indicators are added
    if (data.model || data.experimental_providerMetadata) {
      return data;
    }
    return undefined;
  } catch (e) {
    console.warn('Unable to parse step ai output as JSON');
    return undefined;
  }
};
