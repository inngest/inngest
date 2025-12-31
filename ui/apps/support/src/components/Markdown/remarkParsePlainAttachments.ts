import type { Plugin } from "unified";
import { visit } from "unist-util-visit";

/**
 * Remark plugin that parses Plain inline attachments from base64 JSON data URIs
 * and sets data properties for react-markdown components
 */
export const remarkParsePlainAttachments: Plugin = () => {
  return (tree) => {
    visit(tree, "image", (node: any) => {
      // Plain inline attachments are base64 encoded JSON objects
      try {
        if (node.url && node.url.match(/^data:application\/json/)) {
          const [, encodedData] = node.url.split(",");
          const decodedData = atob(encodedData);
          const json = JSON.parse(decodedData) as {
            attachmentId: string;
            width: string | null;
            height: string | null;
          };
          // Store custom data in the node's data field
          if (!node.data) {
            node.data = {};
          }
          // Set hProperties for HTML attributes
          node.data.hProperties = {
            "data-attachment-id": json.attachmentId,
            "data-width": json.width,
            "data-height": json.height,
          };
        }
      } catch (err) {
        console.error("Error parsing inline attachment:", err);
      }
    });
  };
};
