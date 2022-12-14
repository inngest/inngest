import IconTheme from "./theme";

export function IconSDK({ size = 20, className = "", color = "transparent" }) {
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
          x="-79.3%"
          y="-56.7%"
          width="258.7%"
          height="258.7%"
          filterUnits="objectBoundingBox"
          id="sdk-a"
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
        <path id="sdk-b" d="M5.2941 3.5294h4.4118v4.4118H5.2941z" />
      </defs>
      <g fill="none" fillRule="evenodd">
        <path d="M0 0h20v20H0z" />
        <g transform="translate(2.5 4.3)" fillRule="nonzero">
          <rect
            fill={theme.color}
            opacity={theme.opacity}
            x="1.7647"
            width="11.4706"
            height="11.4706"
            rx="2"
          />
          <path
            d="M13.2353 2.647h.8823c.4874 0 .8824.3951.8824.8824a.8824.8824 0 0 1-.8824.8824h-.8823V2.647Zm0 2.6471h.8823c.4874 0 .8824.395.8824.8824a.8824.8824 0 0 1-.8824.8823h-.8823V5.2941Zm0 2.647h.8823c.4874 0 .8824.3951.8824.8824a.8824.8824 0 0 1-.8824.8824h-.8823V7.9412ZM.8823 2.6472h.8824v1.7647H.8824a.8824.8824 0 0 1 0-1.7647Zm0 2.647h.8824v1.7647H.8824a.8824.8824 0 0 1 0-1.7647Zm0 2.647h.8824V9.706H.8824a.8824.8824 0 0 1 0-1.7647Z"
            fill="#FFF"
          />
          <use fill="#000" filter="url(#sdk-a)" xlinkHref="#sdk-b" />
          <use fill="#FFF" xlinkHref="#sdk-b" />
        </g>
      </g>
    </svg>
  );
}
