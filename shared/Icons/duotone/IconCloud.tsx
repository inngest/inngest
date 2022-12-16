import IconTheme from "./theme";
export function IconCloud({
  size = 20,
  className = "",
  color = "transparent",
}) {
  const theme = IconTheme(color);
  return (
    <svg
      width={size}
      height={size}
      className={className}
      viewBox="0 0 20 20"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="none" fillRule="nonzero">
        <path
          d="M6.3513 12.1552c-1.2143-.757-2.018-2.0739-2.018-3.5719 0-2.3472 1.9733-4.25 4.4074-4.25 2.0537 0 3.7793 1.3545 4.2686 3.1875h1.7916c1.5214 0 2.7547 1.1893 2.7547 2.6563 0 1.467-1.2333 2.6562-2.7547 2.6562H8.19c-.7066 0-1.351-.2565-1.8386-.6781Z"
          fill={theme.color}
          opacity={theme.opacity}
        />
        <path
          d="M3.518 14.9885C2.3037 14.2316 1.5 12.9146 1.5 11.4167c0-2.3472 1.9733-4.25 4.4074-4.25 2.0537 0 3.7793 1.3544 4.2686 3.1875h1.7916c1.5213 0 2.7546 1.1892 2.7546 2.6562s-1.2333 2.6563-2.7546 2.6563H5.3565c-.7065 0-1.351-.2565-1.8385-.6782Z"
          fill="#FFF"
        />
      </g>
    </svg>
  );
}
