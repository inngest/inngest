import { ComponentChildren } from "preact";
import { useEffect } from "preact/hooks";

interface CodeBlockModalProps {
  children: ComponentChildren;
  closeModal: () => void;
}

export default function CodeBlockModal({
  children,
  closeModal,
}: CodeBlockModalProps) {
  useEffect(() => {
    const close = (e) => {
      if (e.key === "Escape") {
        closeModal();
      }
    };
    window.addEventListener("keydown", close);
    return () => window.removeEventListener("keydown", close);
  }, []);

  return (
    <div
      className="fixed inset-0 z-50 px-6 py-6 flex items-center justify-center bg-black/50 w-screen h-screen"
      >
        {children}
      </div>
    </div>
  );
}
