import { ElementWrapper, TextElement } from '../DetailsCard/Element';
import { getAIInfo, type ExperimentalAI } from './utils';

export const AITrace = ({ aiOutput }: { aiOutput?: ExperimentalAI }) => {
  if (!aiOutput) {
    return null;
  }
  const { promptTokens, completionTokens, totalTokens, model } = getAIInfo(aiOutput);

  //
  // upstream parsing is quite forgiving,
  // only show ai metadata it actually exists
  return (
    <>
      {model && (
        <ElementWrapper label="Model">
          <TextElement>{model}</TextElement>
        </ElementWrapper>
      )}
      {typeof promptTokens === 'number' && (
        <ElementWrapper label="Prompt Tokens">
          <TextElement>{promptTokens}</TextElement>
        </ElementWrapper>
      )}
      {typeof completionTokens === 'number' && (
        <ElementWrapper label="Completion Tokens">
          <TextElement>{completionTokens}</TextElement>
        </ElementWrapper>
      )}
      {typeof totalTokens === 'number' && (
        <ElementWrapper label="Total Tokens">
          <TextElement>{totalTokens}</TextElement>
        </ElementWrapper>
      )}
    </>
  );
};
