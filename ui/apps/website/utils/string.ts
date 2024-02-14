/**
 * Given a string, removes indentation in a predicatable manner.
 *
 * If `greedy` is `true`, then it'll accept an indentation of 0 as the smallest
 * it finds. This isn't _usually_ what you want, but it fits certain use cases.
 *
 * @example
 * ```ts
 * stripIndent(`This string
 *   should only indent
 *     right here
 *   but nowhere else,
 *   as it's obvious we don't want
 *   the other indentations`);
 *
 * // This string
 * // should only indent
 * //   right here
 * // but nowhere else
 * // as it's obvious we don't want
 * // the other indentations
 *
 * stripIndent(`This line will be matched as the smallest,
 *   so this line will be indented.`);
 *
 * // This line will be matched as the smallest
 * //   so this line will be indented.
 * ```
 */
export const stripIndent = (str: string, greedy?: boolean) => {
  const matchRe = new RegExp(`^[ \\t]${greedy ? "*" : "+"}(?=\\S)`, "gm");
  const match = str.match(matchRe);

  if (!match) {
    return str;
  }

  const indent = Math.min(...match.map((x) => x.length));
  const re = new RegExp(`^[ \\t]{${indent}}`, "gm");

  return indent > 0 ? str.replace(re, "") : str;
};
