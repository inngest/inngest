import visit from 'unist-util-visit';

// Usage:
//
// serialize(content, {
//   mdxOptions: {
//     remarkPlugins: [await createHighlightPlugin()],
//   },
// });
export const highlightPlugin = (() => {
  // memoization.
  let shiki: any;
  let highlighter: any;

  // This is an IIFE, and so createHihlightPlugin
  return async (): Promise<any> => {
    if (!shiki || !highlighter) {
      shiki = await import('shiki');
      highlighter = await shiki.getHighlighter({ theme: "nord" });
    }
    return [doHighlight, { highlighter }];
  };
})()

const doHighlight = (options: any) => async (tree: any) => {
	visit(tree, 'code', (node: any) => {
		node.type = 'html'
		node.children = undefined
		node.value = options.highlighter
      .codeToHtml(node.value, node.lang)
      .replace('<pre class="shiki"', `<pre class="shiki" language="${node.lang}" meta="${node.meta}"`)
	})
}

