import type { Plugin } from "unified";
import type { Root } from "hast";
import { visit } from "unist-util-visit";

/**
 * Rehype plugin that preserves hProperties from mdast nodes to hast nodes
 * This ensures data attributes set in remark plugins are available in React components
 */
export const rehypePreserveHProperties: Plugin<[], Root> = () => {
  return (tree) => {
    visit(tree, "element", (node: any) => {
      // If the node has hProperties from the mdast phase, apply them to the element
      if (node.data?.hProperties) {
        node.properties = {
          ...node.properties,
          ...node.data.hProperties,
        };
      }
    });
  };
};
