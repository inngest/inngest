import IconTheme from "./theme";

export function IconSteps({
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
          d="M14.2376 16c1.3645 0 2.4706-1.1061 2.4706-2.4706s-1.106-2.4706-2.4706-2.4706c-1.3644 0-2.4705 1.1061-2.4705 2.4706S12.873 16 14.2376 16ZM5.7671 8.9412c1.3644 0 2.4705-1.1061 2.4705-2.4706S7.1316 4 5.7671 4C4.4026 4 3.2965 5.1061 3.2965 6.4706s1.106 2.4706 2.4706 2.4706Z"
          fill="#FFF"
        />
        <path
          d="M16.3553 7H12.12a.5294.5294 0 0 1 0-1.0588h4.2353a.5294.5294 0 1 1 0 1.0588Zm-8.4706 7.0588H3.6494a.5294.5294 0 1 1 0-1.0588h4.2353a.5294.5294 0 1 1 0 1.0588Z"
          fill={theme.color}
          opacity={theme.opacity}
        />
      </g>
    </svg>
  );
}
