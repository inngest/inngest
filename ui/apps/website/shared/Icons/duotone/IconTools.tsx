import IconTheme, { IconProps } from "./theme";

export function IconTools({
  size = 20,
  className = "",
  color = "transparent",
}: IconProps) {
  const theme = IconTheme(color);
  return (
    <svg
      width={size}
      height={size}
      className={className}
      viewBox="0 0 20 20"
      xmlns="http://www.w3.org/2000/svg"
      xmlnsXlink="http://www.w3.org/1999/xlink"
    >
      <defs>
        <filter
          x="-32.8%"
          y="-28.8%"
          width="165.6%"
          height="180.8%"
          filterUnits="objectBoundingBox"
          id="tools-a"
        >
          <feOffset dy="1" in="SourceAlpha" result="shadowOffsetOuter1" />
          <feGaussianBlur
            stdDeviation="1"
            in="shadowOffsetOuter1"
            result="shadowBlurOuter1"
          />
          <feColorMatrix
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.268793706 0"
            in="shadowBlurOuter1"
          />
        </filter>
        <path
          d="m9.98 6.505 1.2297 4.9183.5273-1.0548A.6667.6667 0 0 1 12.3333 10h2.3334a.6667.6667 0 0 1 0 1.3333h-1.9213l-1.1491 2.2982c-.2768.5536-1.093.464-1.243-.1365L9.2417 9.0494l-.6093 1.828a.6667.6667 0 0 1-.6325.456H5.3333a.6667.6667 0 1 1 0-1.3334h2.1862l1.1814-3.5442c.2109-.6327 1.1174-.5979 1.2792.0492Zm4.6867-1.1717a.6667.6667 0 1 1 0 1.3334.6667.6667 0 0 1 0-1.3334Z"
          id="tools-b"
        />
      </defs>
      <g fill="none" fillRule="nonzero">
        <rect
          fill={theme.color}
          opacity={theme.opacity}
          x="3.3333"
          y="4"
          width="13.3333"
          height="12"
          rx="2"
        />
        <use fill="#000" filter="url(#tools-a)" xlinkHref="#tools-b" />
        <use fill="#FFF" xlinkHref="#tools-b" />
      </g>
    </svg>
  );
}
