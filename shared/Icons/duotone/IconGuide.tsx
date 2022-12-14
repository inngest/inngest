import IconTheme from "./theme";

export function IconGuide({
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
      <defs>
        <filter
          x="-190.3%"
          y="-22.8%"
          width="480.5%"
          height="163.8%"
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
          d="M13.0671 4.5027h.017a.9.9 0 0 1 .902.9018l-.019 9.1602a.9.9 0 0 1-.898.8982h-.0171a.9.9 0 0 1-.9019-.9018l.0189-9.1603a.9.9 0 0 1 .8981-.8981Z"
          id="guide-b"
        />
      </defs>
      <g fill="none" fill-rule="nonzero">
        <path
          d="M5.3053 4.532h.6052c.3343 0 .6053.2728.6053.6093v9.7494c0 .3365-.271.6093-.6053.6093h-.6052c-.3343 0-.6053-.2728-.6053-.6093V5.1413c0-.3365.271-.6093.6053-.6093Zm3.0263 0h.6052c.3343 0 .6053.2728.6053.6093v9.7494c0 .3365-.271.6093-.6053.6093h-.6052c-.3343 0-.6053-.2728-.6053-.6093V5.1413c0-.3365.271-.6093.6053-.6093Z"
          fill={theme.color}
          opacity={theme.opacity}
        />
        <g transform="rotate(-19 13.068 9.9828)">
          <use fill="#000" filter="url(#guide-a)" xlinkHref="#guide-b" />
          <use fill="#FFF" xlinkHref="#guide-b" />
        </g>
      </g>
    </svg>
  );
}
