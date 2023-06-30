import { Highlight, themes } from 'prism-react-renderer';

interface SyntaxHighlightProps {
  code: string;
  language?: 'js';
  className?: string;
}

export const SyntaxHighlight = ({ code, language = 'js' }: SyntaxHighlightProps) => {
  let value = code;
  if (typeof code !== 'string') {
    value = JSON.stringify(code);
  }

  return (
    <Highlight theme={themes.palenight} code={value} language={language}>
      {({ style, tokens, getLineProps, getTokenProps }) => (
        <pre style={style}>
          {tokens.map((line, i) => (
            <div key={i} {...getLineProps({ line })}>
              {line.map((token, key) => (
                <span key={key} {...getTokenProps({ token })} />
              ))}
            </div>
          ))}
        </pre>
      )}
    </Highlight>
  );
};
