import IconTheme from "./theme";

export function IconDeploying({
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
          x="-50%"
          y="-53.6%"
          width="200%"
          height="250%"
          filterUnits="objectBoundingBox"
          id="deploying-a"
        >
          <feOffset dy="2" in="SourceAlpha" result="shadowOffsetOuter1" />
          <feGaussianBlur
            stdDeviation="2"
            in="shadowOffsetOuter1"
            result="shadowBlurOuter1"
          />
          <feColorMatrix
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.5 0"
            in="shadowBlurOuter1"
          />
        </filter>
        <filter
          x="-64.2%"
          y="-46.3%"
          width="228.3%"
          height="229.8%"
          filterUnits="objectBoundingBox"
          id="deploying-c"
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
          d="M5.1367 11.9787C3.851 11.1476 3 9.7015 3 8.0567 3 5.4793 5.0893 3.39 7.6667 3.39c2.1744 0 4.0016 1.4872 4.5196 3.5h1.897C15.6942 6.89 17 8.1958 17 9.8067c0 1.6108-1.3058 2.9166-2.9167 2.9166h-7a2.9058 2.9058 0 0 1-1.9466-.7446Z"
          id="deploying-b"
        />
        <path
          d="M9.223 11.5763v1.5217H7.6615a.3889.3889 0 0 0-.389.3889v.7959c0 .2147.1742.3889.389.3889H9.223v1.5217a.3889.3889 0 0 0 .6402.2968l2.7266-2.3086a.3889.3889 0 0 0 0-.5936l-2.7266-2.3085a.3889.3889 0 0 0-.6402.2968Z"
          id="deploying-d"
        />
      </defs>
      <g fill="none" fill-rule="nonzero">
        <use fill="#000" filter="url(#deploying-a)" xlinkHref="#deploying-b" />
        <use
          fill={theme.color}
          opacity={theme.opacity}
          xlinkHref="#deploying-b"
        />
        <g transform="rotate(-90 10 13.8848)">
          <use
            fill="#000"
            filter="url(#deploying-c)"
            xlinkHref="#deploying-d"
          />
          <use fill="#FFF" xlinkHref="#deploying-d" />
        </g>
      </g>
    </svg>
  );
}
