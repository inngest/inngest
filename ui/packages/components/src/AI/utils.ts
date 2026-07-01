export type OpenAIOutput = {
  model: string;
  experimental_providerMetadata?: never;
  response?: never;
} & Object;

export type VercelAIOutput = {
  experimental_providerMetadata: unknown;
  model?: never;
  response?: never;
} & Object;

export type GoogleAIOutput = {
  response: {
    modelVersion?: string;
  };
  model?: never;
  experimental_providerMetadata: unknown;
} & Object;

export type ExperimentalAI = OpenAIOutput | VercelAIOutput | GoogleAIOutput;

export type AIInfo = {
  inputTokens?: number;
  outputTokens?: number;
  totalTokens?: number;
  model?: string;
};

type MetadataLike = {
  kind?: string | null;
  scope?: string | null;
  updatedAt?: string | null;
  values?: Record<string, unknown> | null;
};

type Value = string | number | boolean | null;
type Object = { [key: string]: Value | Object | Array<Value | Object> };

const AI_METADATA_KIND = 'inngest.ai';
const preferredMetadataScopes = ['step_attempt', 'extended_trace'] as const;

const isObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

const asNumber = (value: unknown): number | undefined =>
  typeof value === 'number' && Number.isFinite(value) ? value : undefined;

const asString = (value: unknown): string | undefined =>
  typeof value === 'string' && value.length > 0 ? value : undefined;

const withComputedTotalTokens = (info: AIInfo): AIInfo => {
  if (
    typeof info.totalTokens !== 'number' &&
    typeof info.inputTokens === 'number' &&
    typeof info.outputTokens === 'number'
  ) {
    return {
      ...info,
      totalTokens: info.inputTokens + info.outputTokens,
    };
  }

  return info;
};

const hasAIInfo = (info: AIInfo | undefined): info is AIInfo =>
  Boolean(
    info &&
      (typeof info.inputTokens === 'number' ||
        typeof info.outputTokens === 'number' ||
        typeof info.totalTokens === 'number' ||
        info.model)
  );

const getModelFromStep = (step: unknown): string | undefined => {
  if (!isObject(step)) {
    return undefined;
  }

  const response = isObject(step.response) ? step.response : undefined;
  if (response) {
    const responseModel = asString(response.modelId);
    if (responseModel) {
      return responseModel;
    }
  }

  const request = isObject(step.request) ? step.request : undefined;
  const body = request && isObject(request.body) ? request.body : undefined;
  return body ? asString(body.model) : undefined;
};

const getUsageInfo = (usage: unknown): AIInfo | undefined => {
  if (!isObject(usage)) {
    return undefined;
  }

  const candidates: AIInfo[] = [
    {
      inputTokens: asNumber(usage.prompt_tokens),
      outputTokens: asNumber(usage.completion_tokens),
      totalTokens: asNumber(usage.total_tokens),
    },
    {
      inputTokens: asNumber(usage.input_tokens),
      outputTokens: asNumber(usage.output_tokens),
      totalTokens: asNumber(usage.total_tokens),
    },
    {
      inputTokens: asNumber(usage.promptTokens),
      outputTokens: asNumber(usage.completionTokens),
      totalTokens: asNumber(usage.totalTokens),
    },
    {
      inputTokens: asNumber(usage.inputTokens),
      outputTokens: asNumber(usage.outputTokens),
      totalTokens: asNumber(usage.totalTokens),
    },
    {
      inputTokens: asNumber(usage.input),
      outputTokens: asNumber(usage.output),
      totalTokens: asNumber(usage.totaltokens),
    },
  ];

  return candidates.map(withComputedTotalTokens).find(hasAIInfo);
};

const getGoogleAIInfo = (obj: Object): AIInfo | undefined => {
  const response = isObject(obj.response) ? obj.response : undefined;
  const usageMetadata =
    response && isObject(response.usageMetadata) ? response.usageMetadata : undefined;

  if (!response || !usageMetadata) {
    return undefined;
  }

  return withComputedTotalTokens({
    model: asString(response.modelVersion),
    inputTokens: asNumber(usageMetadata.promptTokenCount),
    outputTokens: asNumber(usageMetadata.candidatesTokenCount),
    totalTokens: asNumber(usageMetadata.totalTokenCount),
  });
};

const isLikelyVercelOutput = (obj: Object): boolean => {
  const response = isObject(obj.response) ? obj.response : undefined;

  return Boolean(
    obj.experimental_providerMetadata ||
      obj.totalUsage ||
      obj.rawResponse ||
      (Array.isArray(obj.steps) && obj.steps.length > 0) ||
      (response && response.modelId)
  );
};

const getVercelAIInfo = (obj: Object): AIInfo | undefined => {
  if (!isLikelyVercelOutput(obj)) {
    return undefined;
  }

  const totalUsageInfo = getUsageInfo(obj.totalUsage);
  const topLevelUsageInfo = getUsageInfo(obj.usage);

  const steps = Array.isArray(obj.steps) ? obj.steps : [];
  const firstStep = steps.find((step): step is Object => isObject(step));
  const stepUsageInfo = firstStep ? getUsageInfo(firstStep.usage) : undefined;

  const model =
    asString(obj.modelId) ??
    (isObject(obj.response) ? asString(obj.response.modelId) : undefined) ??
    (firstStep ? getModelFromStep(firstStep) : undefined);

  const usageInfo = totalUsageInfo ?? topLevelUsageInfo ?? stepUsageInfo;
  if (!usageInfo && !model) {
    return undefined;
  }

  return withComputedTotalTokens({
    ...(usageInfo ?? {}),
    model,
  });
};

export const parseAIOutput = (output: string): ExperimentalAI | undefined => {
  try {
    const data = JSON.parse(output);

    //
    // infer run output is on body
    // TODO: use proper trace infer indicator
    if (data.body) {
      return data.body;
    }

    //
    // infer run step data is on data.data
    // TODO: use proper trace infer indicator
    if (data.data) {
      return data.data;
    }

    if (data.model || data.experimental_providerMetadata || data.response) {
      return data;
    }
    return undefined;
  } catch (e) {
    console.warn('Unable to parse step ai output as JSON');
    return undefined;
  }
};

export const getAIInfo = (obj: Object): AIInfo => {
  if (isObject(obj.body)) {
    return getAIInfo(obj.body as Object);
  }

  if (isObject(obj.data)) {
    return getAIInfo(obj.data as Object);
  }

  const directUsageInfo = getUsageInfo(obj.usage);
  const directInfo = withComputedTotalTokens({
    model: asString(obj.model),
    ...(directUsageInfo ?? {}),
  });

  const info = getGoogleAIInfo(obj) ?? getVercelAIInfo(obj) ?? directInfo;

  return hasAIInfo(info) ? info : {};
};

export const getAIInfoFromMetadata = (metadata?: MetadataLike[] | null): AIInfo | undefined => {
  if (!metadata?.length) {
    return undefined;
  }

  for (const scope of preferredMetadataScopes) {
    const match = metadata
      .filter((entry): entry is MetadataLike =>
        Boolean(
          entry &&
            entry.kind === AI_METADATA_KIND &&
            entry.scope === scope &&
            isObject(entry.values)
        )
      )
      .sort((a, b) => {
        const aTime = a.updatedAt ? new Date(a.updatedAt).getTime() : 0;
        const bTime = b.updatedAt ? new Date(b.updatedAt).getTime() : 0;
        return bTime - aTime;
      })[0];

    if (!match || !match.values) {
      continue;
    }

    const info = withComputedTotalTokens({
      inputTokens: asNumber(match.values.input_tokens),
      outputTokens: asNumber(match.values.output_tokens),
      totalTokens: asNumber(match.values.total_tokens),
      model: asString(match.values.model),
    });

    if (hasAIInfo(info)) {
      return info;
    }
  }

  return undefined;
};
