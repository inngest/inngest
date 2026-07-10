import { ElementWrapper, TextElement } from '../DetailsCard/Element';
import type { AIInfo } from './utils';

export const AITrace = ({ aiInfo }: { aiInfo?: AIInfo }) => {
  if (!aiInfo) {
    return null;
  }
  const { inputTokens, outputTokens, totalTokens, model } = aiInfo;

  return (
    <>
      {model && (
        <ElementWrapper label="Model">
          <TextElement>{model}</TextElement>
        </ElementWrapper>
      )}
      {typeof inputTokens === 'number' && (
        <ElementWrapper label="Input Tokens">
          <TextElement>{inputTokens}</TextElement>
        </ElementWrapper>
      )}
      {typeof outputTokens === 'number' && (
        <ElementWrapper label="Output Tokens">
          <TextElement>{outputTokens}</TextElement>
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
