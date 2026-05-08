export const ExperimentsIcon = ({ className }: { className?: string }) => {
  return (
    <svg
      className={className}
      fill="none"
      height="18"
      viewBox="0 0 18 18"
      width="18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M6.75 2.25V6.44L3.09 12.75C2.46 13.83 3.24 15.18 4.47 15.18H13.53C14.76 15.18 15.54 13.83 14.91 12.75L11.25 6.44V2.25M5.625 2.25H12.375"
        stroke="currentColor"
        strokeWidth="1.35"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx="7.5" cy="11.25" r="1.125" fill="currentColor" />
      <circle cx="10.875" cy="12.75" r="0.75" fill="currentColor" />
    </svg>
  );
};
