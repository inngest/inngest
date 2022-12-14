import IconTheme from "./theme";

export function IconJourney({
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
      xmlnsXlink="http://www.w3.org/1999/xlink"
    >
      <defs>
        <filter
          x="-26.9%"
          y="-19.2%"
          width="153.8%"
          height="153.8%"
          filterUnits="objectBoundingBox"
          id="a"
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
          d="M6.1528 3.858c-.2824-.4773-.982-.4773-1.2645 0L3.6008 6.0339c-.2846.481.0673 1.0852.6323 1.0852h2.575c.5649 0 .9169-.6042.6322-1.0852L6.1528 3.858Zm5.9554 10.477c0-1.1955.9831-2.1647 2.196-2.1647 1.2127 0 2.1958.9692 2.1958 2.1648 0 1.1955-.9831 2.1648-2.1959 2.1648s-2.1959-.9693-2.1959-2.1648Z"
          id="journey-b"
        />
      </defs>
      <g fill="none" fill-rule="nonzero">
        <path
          d="M12.1082 4.9545c-.8085 0-1.4639.6461-1.4639 1.4432v7.216c0 1.594-1.3109 2.8863-2.9279 2.8863-1.617 0-2.9278-1.2923-2.9278-2.8864V5.6761h1.464v7.9375c0 .797.6553 1.4432 1.4638 1.4432.8086 0 1.464-.6461 1.464-1.4432v-7.216c0-1.594 1.3108-2.8863 2.9278-2.8863s2.9279 1.2923 2.9279 2.8864v7.216h-1.464v-7.216c0-.797-.6554-1.4432-1.4639-1.4432Z"
          fill={theme.color}
          opacity={theme.opacity}
        />
        <use fill="#000" filter="url(#journey-a)" xlinkHref="#journey-b" />
        <use fill="#FFF" xlinkHref="#journey-b" />
      </g>
    </svg>
  );
}
