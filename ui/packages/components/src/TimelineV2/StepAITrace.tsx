import type { ExperimentalAI, OpenAIOutput, VercelAIOutput } from '../AI/utils';
import { ElementWrapper, TextElement } from '../DetailsCard/Element';

const isVercelAI = (output: ExperimentalAI): output is VercelAIOutput => {
  return 'experimental_providerMetadata' in output;
};

const isOpenAI = (output: ExperimentalAI): output is OpenAIOutput => {
  return 'model' in output;
};

export const StepAITrace = ({ aiOutput }: { aiOutput?: ExperimentalAI }) =>
  !aiOutput ? null : isVercelAI(aiOutput) ? (
    <VercelAITrace {...aiOutput} />
  ) : (
    <OpenAITrace {...aiOutput} />
  );

const OpenAITrace = ({ model, usage }: OpenAIOutput) => (
  <>
    <ElementWrapper label="Model">
      <TextElement>{model}</TextElement>
    </ElementWrapper>
    <ElementWrapper label="Prompt Tokens">
      <TextElement>{usage.prompt_tokens}</TextElement>
    </ElementWrapper>
    <ElementWrapper label="Completion Tokens">
      <TextElement>{usage.completion_tokens}</TextElement>
    </ElementWrapper>
    <ElementWrapper label="Total Tokens">
      <TextElement>{usage.total_tokens}</TextElement>
    </ElementWrapper>
  </>
);

const VercelAITrace = ({ roundtrips, usage }: VercelAIOutput) => (
  <>
    <ElementWrapper label="Model">
      <TextElement>{roundtrips[0].response.modelId}</TextElement>
    </ElementWrapper>
    <ElementWrapper label="Prompt Tokens">
      <TextElement>{usage.promptTokens}</TextElement>
    </ElementWrapper>
    <ElementWrapper label="Completion Tokens">
      <TextElement>{usage.completionTokens}</TextElement>
    </ElementWrapper>
    <ElementWrapper label="Total Tokens">
      <TextElement>{usage.totalTokens}</TextElement>
    </ElementWrapper>
  </>
);
