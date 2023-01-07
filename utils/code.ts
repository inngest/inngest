import shiki from "shiki";
import { visit } from "unist-util-visit";
import { u } from "unist-builder";

// This function is used to hide twoslash code for the time being
export function rehypeRemoveTwoSlashMarkup() {
  return async (tree) => {
    visit(tree, "element", (node, _nodeIndex, parentNode) => {
      // Get all the code blocks, not the inline <code> `` blocks
      if (node.tagName === "pre" && node.children[0]?.tagName === "code") {
        let codeNode = node.children?.[0];
        let textNode = codeNode?.children?.[0];

        // Remove all code before the cut line
        const cut = textNode.value.split("---cut---\n");
        const code = cut.length > 1 ? cut.pop() : cut[0];
        // Remove comment queries: "//   ^?"
        const removeQueries = code.replace(/\/\/\s+\^\?\s/m, "ok");
        const removeErrorStatements = removeQueries.replace(
          /\/\/ [@errors|@noErrors].+\n/m,
          ""
        );

        textNode.value = removeErrorStatements;
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

    // Enable some custom error highlighting that matches Twoslash
    const ErrorLineRegex = /\/\/ Error: /;
    if (line?.[0]?.content.match(ErrorLineRegex)) {
      console.log(line?.[0]?.content);
      const clean = line?.[0]?.content.replace(ErrorLineRegex, "");
      const errorNode = u(
        "element",
        {
          tagName: "div",
          properties: { class: "error" },
        },
        [u("text", clean)]
      );
      tree.push(errorNode);
      continue;
    }

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
