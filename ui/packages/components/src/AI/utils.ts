const isLLMResponse = (candidate: unknown): boolean => {
  if (typeof candidate !== 'object' || candidate === null) {
    return false;
  }

  const obj = candidate as Record<string, unknown>;

  if (obj.model || obj.experimental_providerMetadata) {
    return true;
  }

  if (typeof obj.usage === 'object' && obj.usage !== null) {
    return true;
  }

  if (
    typeof obj.response === 'object' &&
    obj.response !== null &&
    (obj.response as Record<string, unknown>).modelVersion
  ) {
    return true;
  }

  return false;
};

export const looksLikeAIOutput = (output: string): boolean => {
  try {
    const data = JSON.parse(output);
    return [data, data?.body, data?.data].some(isLLMResponse);
  } catch (e) {
    return false;
  }
};
