import Refractor from 'react-refractor';
/**
 * Load specific languages from refractor.
 * We purposefully load these individually to reduce the bundle size.
 */
import json from 'refractor/lang/json';

/**
 * Highlighting has no styling by default; we use a custom theme here, which is
 * just a Prism theme with some minor modifications.
 */
import './highlight.min.css';

Refractor.registerLanguage(json);

interface SyntaxHighlightProps {
  /**
   * The code to highlight.
   */
  code: string;

  /**
   * The language to highlight the code in.
   *
   * Defaults to `"json"`.
   */
  language?: 'json';

  /**
   * Any additional classes to add to the root element.
   */
  className?: string;
}

/**
 * Highlight a given `code` string using the given `language`.
 */
export const SyntaxHighlight = ({ code, language = 'json', className }: SyntaxHighlightProps) => {
  // Some types are dyanmically asserted to be string;  this is a
  // safety check in case an API fails to respond with what we expect.
  // Refractor will throw an error and break the UI.
  let value = code;
  if (typeof code !== 'string') {
    value = JSON.stringify(code);
  }

  return <Refractor language={language} value={value} className={className} />;
};
