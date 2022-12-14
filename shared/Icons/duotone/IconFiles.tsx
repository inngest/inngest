import IconTheme from "./theme";
export function IconFiles({
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
      <g fill="none" fill-rule="nonzero">
        <path
          d="M4.5 7h11v8.0357c0 .5326-.4617.9643-1.0313.9643H5.5313C4.9618 16 4.5 15.5683 4.5 15.0357V7Zm4.125 1.9286c-.3797 0-.6875.2878-.6875.6428 0 .355.3078.6429.6875.6429h2.75c.3797 0 .6875-.2878.6875-.6429 0-.355-.3078-.6428-.6875-.6428h-2.75Z"
          fill={theme.color}
          opacity={theme.opacity}
        />
        <rect fill="#FFF" x="3.5" y="4" width="13" height="3" rx="1" />
      </g>
    </svg>
  );
}
