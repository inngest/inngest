import { ElementWrapper, TextElement } from '../DetailsCard/Element';
import { getAIInfo, type ExperimentalAI, type Value } from './utils';

export const AITrace = ({ aiOutput }: { aiOutput?: ExperimentalAI }) => {
  if (!aiOutput) {
    return null;
  }
  const { promptTokens, completionTokens, totalTokens, model } = getAIInfo(aiOutput);

  return (
    <>
      <ElementWrapper label="Model">
        <TextElement>{model}</TextElement>
      </ElementWrapper>
      <ElementWrapper label="Prompt Tokens">
        <TextElement>{promptTokens}</TextElement>
      </ElementWrapper>
      <ElementWrapper label="Completion Tokens">
        <TextElement>{completionTokens}</TextElement>
      </ElementWrapper>
      <ElementWrapper label="Total Tokens">
        <TextElement>{totalTokens}</TextElement>
      </ElementWrapper>
    </>
  );
};
