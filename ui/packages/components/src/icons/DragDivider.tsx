export const DragDivider = ({ className }: { className?: string }) => {
  return (
    <svg
      width="8"
      height="32"
      viewBox="0 0 8 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      <rect
        x="0.5"
        y="0.5"
        width="7"
        height="31"
        rx="3.5"
        className="stroke-muted"
        strokeWidth="1"
      />
      <path d="M4 10V22" className="stroke-subtle" strokeLinecap="round" strokeWidth="1" />
    </svg>
  );
};
