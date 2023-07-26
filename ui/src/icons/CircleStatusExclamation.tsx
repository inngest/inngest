export function IconCircleStatusExclamation({
  className,
  withOutline,
}: {
  className?: string;
  withOutline?: boolean;
}) {
  return (
    <svg
      width="28"
      height="28"
      viewBox="0 0 30 30"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      color="#F43F5E"
    >
      <g>
        {withOutline && (
          <path
            d="M15 29C22.732 29 29 22.732 29 15C29 7.26801 22.732 1 15 1C7.26801 1 1 7.26801 1 15C1 22.732 7.26801 29 15 29Z"
            stroke="#1E293B"
          />
        )}
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
