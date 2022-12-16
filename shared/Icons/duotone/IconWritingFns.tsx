import IconTheme from "./theme";
export function IconWritingFns({
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
          x="-28.1%"
          y="-25.8%"
          width="156.2%"
          height="172.2%"
          filterUnits="objectBoundingBox"
          id="writing-fn-a"
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
        <filter
          x="-53.4%"
          y="-65.9%"
          width="206.8%"
          height="284.4%"
          filterUnits="objectBoundingBox"
          id="writing-fn-c"
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
          d="M3.7884 11.1308V5.609A1.8305 1.8305 0 0 1 5.859 3.5384h8.2826a1.8305 1.8305 0 0 1 2.0707 2.0706v5.5218a1.8305 1.8305 0 0 1-2.0707 2.0706H5.859a1.8305 1.8305 0 0 1-2.0706-2.0706Z"
          id="writing-fn-b"
        />
        <path
          d="M8.6199 10.268a.5135.5135 0 0 1-.3658-.1519L6.8736 8.7357a.517.517 0 0 1 0-.7316l1.3805-1.3805a.5177.5177 0 0 1 .7316.7317l-1.014 1.0146 1.014 1.0146a.5177.5177 0 0 1-.3658.8835Zm3.1267-.1519 1.3804-1.3804a.517.517 0 0 0 0-.7316l-1.3804-1.3805a.5177.5177 0 0 0-.7317.7317l1.014 1.0146-1.014 1.0146a.5177.5177 0 1 0 .7317.7316Z"
          id="writing-fn-d"
        />
      </defs>
      <g fill="none" fillRule="nonzero">
        <use
          fill="#000"
          filter="url(#writing-fn-a)"
          xlinkHref="#writing-fn-b"
        />
        <use
          fill={theme.color}
          opacity={theme.opacity}
          xlinkHref="#writing-fn-b"
        />
        <path
          d="M13.1063 15.4446h-1.7255v-2.2432h-2.761v2.2432H6.8944a.5177.5177 0 1 0 0 1.0353h6.212a.5177.5177 0 1 0 0-1.0353Z"
          fill="#FFF"
        />
        <use
          fill="#000"
          filter="url(#writing-fn-c)"
          xlinkHref="#writing-fn-d"
        />
        <use fill="#FFF" xlinkHref="#writing-fn-d" />
      </g>
    </svg>
  );
}
