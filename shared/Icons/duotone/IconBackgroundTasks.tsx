import IconTheme from "./theme";
export function IconBackgroundTasks({
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
          x="-74.7%"
          y="-49.6%"
          width="249.4%"
          height="238.8%"
          filterUnits="objectBoundingBox"
          id="bg-tasks-a"
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
          d="M4.1667 8.4359V5.5641c-.0393-.2966.0533-.5958.25-.8076.1967-.2119.4746-.3116.75-.2693h2.6666c.2754-.0423.5533.0574.75.2693.1967.2118.2893.511.25.8076v2.8718c.0393.2966-.0533.5958-.25.8076-.1967.2119-.4746.3116-.75.2693H5.1667c-.2754.0423-.5533-.0574-.75-.2693-.1967-.2118-.2893-.511-.25-.8076Z"
          id="bg-tasks-b"
        />
      </defs>
      <g fillRule="nonzero" fill="none">
        <path
          d="M5.3333 12.8718V7.1282c-.0786-.5932.1067-1.1916.5001-1.6153.3935-.4236.9491-.6232 1.5-.5385h5.3333c.5508-.0847 1.1064.1149 1.4999.5385.3934.4237.5787 1.0221.5 1.6153v5.7436c.0787.5932-.1066 1.1916-.5 1.6153-.3935.4236-.9491.6232-1.5.5385H7.3334c-.5508.0847-1.1064-.1149-1.4999-.5385-.3934-.4237-.5787-1.0221-.5-1.6153Z"
          fill={theme.color}
          opacity={theme.opacity}
        />
        <g transform="translate(3.5 3)">
          <use fill="#000" filter="url(#bg-tasks-a)" xlinkHref="#bg-tasks-b" />
          <use fill="#FFF" xlinkHref="#bg-tasks-b" />
        </g>
        <path
          d="M16.5 7.8462c.0002.1428-.0524.2799-.1463.381-.0938.101-.221.1576-.3537.1574h-1.3333v-1.077H16c.1327-.0001.26.0566.3537.1576.0939.101.1465.238.1463.381Zm-.5 3.7692h-1.3333v1.077H16c.2761 0 .5-.2412.5-.5386 0-.2973-.2239-.5384-.5-.5384ZM4 7.3077c-.2761 0-.5.241-.5.5385 0 .2973.2239.5384.5.5384h1.3333v-1.077H4Zm0 4.3077c-.2761 0-.5.241-.5.5384s.2239.5385.5.5385h1.3333v-1.077H4ZM12 3c-.2754.002-.4982.2419-.5.5385v1.4359h1v-1.436c.0002-.1428-.0524-.2799-.1463-.3809C12.26 3.0565 12.1327 2.9998 12 3ZM8 3c-.2754.002-.4982.2419-.5.5385v1.4359h1v-1.436c.0002-.1428-.0524-.2799-.1463-.3809C8.26 3.0565 8.1327 2.9998 8 3Zm3.5 12.0256v1.436c0 .2973.2239.5384.5.5384s.5-.241.5-.5385v-1.4359h-1Zm-4 0v1.436c0 .2973.2239.5384.5.5384s.5-.241.5-.5385v-1.4359h-1Z"
          fill="#FFF"
        />
      </g>
    </svg>
  );
}
