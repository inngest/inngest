import shiki from "shiki";
import { visit } from "unist-util-visit";
import { u } from "unist-builder";

export function rehypePrependCode() {
  return (tree) => {
    visit(tree, "element", (node, _nodeIndex, parentNode) => {
      if (node.tagName === "code" && node.properties.className) {
        // parentNode.properties.language = node.properties.className[0]?.replace(
        // /^language-/,
        // ""
        // );
      }
    });
  };
}

let highlighter;

// Legacy visitor
export function rehypeShiki() {
  return async (tree) => {
    highlighter =
      highlighter ?? (await shiki.getHighlighter({ theme: "css-variables" }));

    visit(tree, "element", (node) => {
      if (node.tagName === "pre" && node.children[0]?.tagName === "code") {
        let codeNode = node.children[0];
        let textNode = codeNode.children[0];

        node.properties.code = textNode.value;

        // Match what twoslash did
        node.properties.class = "shiki css-variables";
        node.properties.style =
          "background-color:var(--shiki-color-background);color:var(--shiki-color-text)";

        if (node.properties.language) {
          let tokens = highlighter.codeToThemedTokens(
            textNode.value,
            node.properties.language
          );

          // We modify this to replicate what twoslash used to do to the DOM structure
          const tree = tokensToHast(tokens);
          node.children = [
            u(
              "element",
              {
                tagName: "div",
                properties: { class: "language-id" },
              },
              [u("text", node.properties.language)]
            ),
            u(
              "element",
              {
                tagName: "div",
                properties: { class: "code-container" },
              },
              tree
            ),
          ];
        }
      }
    });
  };
}
// Modified from https://github.com/rsclarke/rehype-shiki to generate hast nodes like Twoslash used to
function tokensToHast(lines) {
  let tree = [];

  for (const line of lines) {
    const children = [];

    for (const token of line) {
      children.push(
        u(
          "element",
          {
            tagName: "span",
            properties: { style: "color: " + token.color },
          },
          [u("text", token.content)]
        )
      );
    }

    // Twoslash used to nest things with a "line" div
    const lineNode = u(
      "element",
      {
        tagName: "div",
        properties: { class: "line" },
      },
      children
    );
    tree.push(lineNode);
  }

  // Remove the last blank line
  tree.pop();

  return tree;
}
