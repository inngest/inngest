export default function ClockIcon({
  size = 14,
  className = '',
}: {
  size?: number;
  className?: string;
}) {
  return (
    <svg width="14" height="14" xmlns="http://www.w3.org/2000/svg" className={className}>
      <path
        d="M7 3v4h3m3 0A6 6 0 0 1 1 7c0-3.3137 2.6863-6 6-6s6 2.6863 6 6h0Z"
        stroke="#64748B"
        strokeWidth="1.5"
        fill="none"
        fillRule="evenodd"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
