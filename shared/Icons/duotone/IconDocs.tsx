import IconTheme from "./theme";

export function IconDocs({ size = 20, className = "", color = "transparent" }) {
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
          x="-58.3%"
          y="-24.5%"
          width="216.7%"
          height="168.6%"
          filterUnits="objectBoundingBox"
          id="docs-a"
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
          d="M10.957 15.0735c1.484-.6603 3.3316-1.083 5.543-1.2683v-8.534c0-.205-.1492-.3712-.3333-.3712l-.0085.0001c-1.886.0532-3.7721.5466-5.6582 1.4802v8.3484c0 .205.1492.3713.3333.3713a.3038.3038 0 0 0 .1237-.0265Z"
          id="docs-b"
        />
      </defs>
      <g fill="none" fill-rule="nonzero">
        <use fill="#000" filter="url(#docs-a)" xlinkHref="#docs-b" />
        <use fill="#FFF" xlinkHref="#docs-b" />
        <path
          d="M9.043 15.0735c-1.484-.6603-3.3316-1.083-5.543-1.2683v-8.534c0-.205.1492-.3712.3333-.3712l.0085.0001c1.886.0532 3.7721.5466 5.6582 1.4802v8.3484c0 .205-.1492.3713-.3333.3713a.3038.3038 0 0 1-.1237-.0265Z"
          fill={theme.color}
          opacity={theme.opacity}
        />
      </g>
    </svg>
  );
}
