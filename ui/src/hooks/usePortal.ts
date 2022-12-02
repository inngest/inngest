import type { VNode } from "preact";
import { createPortal } from "preact/compat";

export const usePortal = () => {
  const container = document.getElementById("modals");

  return (node: VNode<{}>) =>
    container ? createPortal(node, container) : null;
};
