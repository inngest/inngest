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

//
// regex pattern to match the key pieces of ai information in outputs:
// promptTokens, completionTokens, totalTokens, model and common variations
// such as promptTokens, prompt_tokens, modelId, model, etc.
const pattern =
  /\b(?:prompt|input|completion|output|total|model)[ _]?(?:id|version|tokens?|TokenCount)?\b|\b(?:prompt|candidates|total)[ _]?(?:tokens?|TokenCount)\b/i;

export type Value = string | number | boolean | null;
type Object = { [key: string]: Value | Object | Array<Value | Object> };

type ResultType = {
  promptTokens?: Value;
  completionTokens?: Value;
  totalTokens?: Value;
  model?: Value;
};

const toNumber = (input?: Value): number => (isNaN(Number(input)) ? 0 : Number(input));

/*
 * recursively search through the object to find any of the key pieces of ai information
 * we care about. For now just take the first match we find for each and stop there.
 */
export const getAIInfo = (obj: Object): ResultType => {
  const info = Object.keys(obj).reduce<ResultType>((acc, key) => {
    if (acc.promptTokens && acc.completionTokens && acc.totalTokens && acc.model) {
      return acc;
    }

    const value = obj[key];

    //
    // Handle arrays by reducing each element into a combined result
    if (Array.isArray(value)) {
      const arrayResult = value.reduce<ResultType>((arrayAcc, item) => {
        if (typeof item === 'object' && item !== null) {
          return { ...arrayAcc, ...getAIInfo(item) };
        }
        return arrayAcc;
      }, {});
      return { ...acc, ...arrayResult };
    }

    if (typeof value === 'object' && value !== null) {
      return { ...acc, ...getAIInfo(value) };
    }

    const match = pattern.exec(key);

    if (match) {
      if (!acc.promptTokens && (/prompt/.test(key) || /input/.test(key))) {
        acc.promptTokens = value;
      } else if (
        !acc.completionTokens &&
        (/completion/.test(key) || /output/.test(key) || /candidatesTokenCount/.test(key))
      ) {
        acc.completionTokens = value;
      } else if (!acc.totalTokens && /total/.test(key)) {
        acc.totalTokens = value;
      } else if (!acc.model && /model/.test(key)) {
        acc.model = value;
      }
    }

    return acc;
  }, {});

  if (!info.totalTokens) {
    info.totalTokens = toNumber(info.promptTokens) + toNumber(info.completionTokens);
  }

  return info;
};
