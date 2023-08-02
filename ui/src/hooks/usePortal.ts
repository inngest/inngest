import { createPortal } from 'react-dom';

export const usePortal = () => {
  let container: HTMLElement | null = null;
  if (typeof window !== 'undefined') {
    container = window.document.getElementById('modals');
  }

  return (node: React.ReactNode) => (container ? createPortal(node, container) : null);
};
