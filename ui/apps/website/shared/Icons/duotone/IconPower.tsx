import IconTheme from "./theme";

export function IconPower({
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
        <circle
          fill={theme.color}
          opacity={theme.opacity}
          cx="10"
          cy="10"
          r="6.5"
        />
        <path
          d="m10.5703 13.2396 1.8806-3.814c.1147-.2327.0217-.5156-.2078-.632a.4592.4592 0 0 0-.2077-.0497h-1.7257V6.971c0-.2601-.208-.471-.4645-.471a.4639.4639 0 0 0-.4155.2604l-1.8806 3.814c-.1147.2327-.0217.5156.2078.632a.4592.4592 0 0 0 .2077.0497h1.7257v1.7729c0 .2601.208.471.4645.471a.4639.4639 0 0 0 .4155-.2604Z"
          fill="#FFF"
        />
      </g>
    </svg>
  );
}
