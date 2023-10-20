export function IconStatusCircleExclamation({
  className,
  title,
}: {
  className?: string;
  title?: string;
}) {
  return (
    <svg
      width="28"
      height="28"
      viewBox="0 0 30 30"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      color="#F43F5E"
      className={className}
    >
      <title>{title}</title>
      <g>
        <path
          d="M15 26C21.0751 26 26 21.0751 26 15C26 8.92487 21.0751 4 15 4C8.92487 4 4 8.92487 4 15C4 21.0751 8.92487 26 15 26Z"
          fill="currentColor"
        />
      </g>
      <text x="12.5" y="20" fill="#FFFFFF" fontSize="15" fontWeight="bold">
        !
      </text>
    </svg>
  );
}
