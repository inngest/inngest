import { createPortal } from "react-dom";

export const usePortal = () => {
  const container = document.getElementById("modals");

  return (node: React.ReactNode) =>
    container ? createPortal(node, container) : null;
};
