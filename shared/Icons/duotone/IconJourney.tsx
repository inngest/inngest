import IconTheme from "./theme";

export function IconJourney({
  size = 20,
  className = "",
  color = "transparent",
}) {
  const theme = IconTheme(color);
  return (
    <svg
      width="20"
      height="20"
      xmlns="http://www.w3.org/2000/svg"
      xmlnsXlink="http://www.w3.org/1999/xlink"
    >
      <defs>
        <filter
          x="-25%"
          y="-17.6%"
          width="150%"
          height="149.3%"
          filterUnits="objectBoundingBox"
          id="journey-a"
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
          d="M5.8569 3.391c-.3042-.5214-1.0576-.5214-1.3618 0L3.1086 5.768c-.3066.5255.0725 1.1854.6809 1.1854h2.773c.6084 0 .9875-.6599.681-1.1854L5.8568 3.391Zm6.4135 11.4452c0-1.3061 1.0587-2.3648 2.3648-2.3648 1.306 0 2.3648 1.0587 2.3648 2.3648 0 1.306-1.0587 2.3648-2.3648 2.3648-1.306 0-2.3648-1.0588-2.3648-2.3648Z"
          id="journey-b"
        />
      </defs>
      <g fill="none" fill-rule="nonzero">
        <path
          d="M12.2704 4.5889c-.8707 0-1.5765.7058-1.5765 1.5765v7.8827c0 1.7413-1.4117 3.153-3.1531 3.153-1.7414 0-3.153-1.4117-3.153-3.153V5.377h1.5765v8.671c0 .8707.7058 1.5765 1.5765 1.5765s1.5765-.7058 1.5765-1.5765V6.1654c0-1.7414 1.4117-3.153 3.153-3.153 1.7415 0 3.1532 1.4116 3.1532 3.153v7.8827h-1.5766V6.1654c0-.8707-.7058-1.5765-1.5765-1.5765Z"
          fill={theme.color}
          opacity={theme.opacity}
        />
        <use fill="#000" filter="url(#journey-a)" xlinkHref="#journey-b" />
        <use fill="#FFF" xlinkHref="#journey-b" />
      </g>
    </svg>
  );
}
