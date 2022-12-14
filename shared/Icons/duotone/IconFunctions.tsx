export function IconFunctions() {
  return (
    <svg width="46" height="46" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <filter
          x="-42.9%"
          y="-42.9%"
          width="185.7%"
          height="185.7%"
          filterUnits="objectBoundingBox"
          id="a"
        >
          <feOffset dy="2" in="SourceAlpha" result="shadowOffsetOuter1" />
          <feGaussianBlur
            stdDeviation="2"
            in="shadowOffsetOuter1"
            result="shadowBlurOuter1"
          />
          <feColorMatrix
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.300726617 0"
            in="shadowBlurOuter1"
            result="shadowMatrixOuter1"
          />
          <feMerge>
            <feMergeNode in="shadowMatrixOuter1" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>
      <g
        filter="url(#a)"
        transform="translate(8 8.8)"
        fill="none"
        fillRule="evenodd"
      >
        <path d="M0 0h28v28H0z" />
        <rect
          fill="#FFF"
          fillRule="nonzero"
          opacity=".3"
          x="4.6667"
          y="4.6667"
          width="18.6667"
          height="18.6667"
          rx="2"
        />
        <path
          fill="#FFF"
          fillRule="nonzero"
          d="M10.5 10.5h7v7h-7zM23.3333 8.1667H24.5c.6443 0 1.1667.5223 1.1667 1.1666 0 .6444-.5224 1.1667-1.1667 1.1667h-1.1667V8.1667ZM23.3333 12.8333H24.5c.6443 0 1.1667.5224 1.1667 1.1667 0 .6443-.5224 1.1667-1.1667 1.1667h-1.1667v-2.3334ZM23.3333 17.5H24.5c.6443 0 1.1667.5223 1.1667 1.1667 0 .6443-.5224 1.1666-1.1667 1.1666h-1.1667V17.5ZM3.5 8.1667h1.1667V10.5H3.5c-.6443 0-1.1667-.5223-1.1667-1.1667 0-.6443.5224-1.1666 1.1667-1.1666ZM3.5 12.8333h1.1667v2.3334H3.5c-.6443 0-1.1667-.5224-1.1667-1.1667 0-.6443.5224-1.1667 1.1667-1.1667ZM3.5 17.5h1.1667v2.3333H3.5c-.6443 0-1.1667-.5223-1.1667-1.1666 0-.6444.5224-1.1667 1.1667-1.1667Z"
        />
      </g>
    </svg>
  );
}
