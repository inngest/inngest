import Refractor from "react-refractor";

/**
 * Load specific languages from refractor.
 * We purposefully load these individually to reduce the bundle size.
 */
import json from "refractor/lang/json";
Refractor.registerLanguage(json);

/**
 * Highlighting has no styling by default; we use a custom theme here, which is
 * just a Prism theme with some minor modifications.
 */
import "./nord.min.css";

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
  language?: "json";

  /**
   * Any additional classes to add to the root element.
   */
  className?: string;
}

/**
 * Highlight a given `code` string using the given `language`.
 */
export const SyntaxHighlight = ({
  code,
  language = "json",
  className,
}: SyntaxHighlightProps) => {
  return <Refractor language={language} value={code} className={className} />;
};
