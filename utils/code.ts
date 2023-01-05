import shiki from "shiki";
import { visit } from "unist-util-visit";

let highlighter;

export function rehypeShiki() {
  return async (tree) => {
    highlighter =
      highlighter ?? (await shiki.getHighlighter({ theme: "css-variables" }));

    visit(tree, "element", (node) => {
      if (node.tagName === "pre" && node.children[0]?.tagName === "code") {
        let codeNode = node.children[0];
        let textNode = codeNode.children[0];

        node.properties.code = textNode.value;

        if (node.properties.language) {
          let tokens = highlighter.codeToThemedTokens(
            textNode.value,
            node.properties.language
          );

          textNode.value = shiki.renderToHtml(tokens, {
            elements: {
              pre: ({ children }) => children,
              code: ({ children }) => children,
              line: ({ children }) => `<span>${children}</span>`,
            },
          });
        }
      }
    });
  };
}
