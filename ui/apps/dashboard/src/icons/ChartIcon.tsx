export const ChartIcon = ({
  width = 84,
  height = 84,
  className,
}: {
  width?: number;
  height?: number;
  className?: string;
}) => (
  <svg
    width={width}
    height={height}
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    className={className}
  >
    <g clipPath="url(#clip0_6858_42709)">
      <rect x="6" y="6" width={width - 12} height={height - 12} rx="8" fill="#DFF5E6" />
      <path
        d="M31.892 30V51.3333H53.8851V54H29.1428V30H31.892ZM52.9132 34.3905L54.8571 36.2761L47.0123 43.8856L42.8886 39.8867L36.9877 45.6095L35.0437 43.7239L42.8886 36.1144L47.0123 40.1133L52.9132 34.3905Z"
        fill="#2C9B63"
      />
      <path d="M6.58966 -16.7183L6.58965 149.48" stroke="url(#paint0_linear_6858_42709)" />
      <path d="M77.738 -62.5503L77.738 108.817" stroke="url(#paint1_linear_6858_42709)" />
      <path d="M178.247 5.77734L-12.2946 5.77734" stroke="url(#paint2_linear_6858_42709)" />
      <path d="M105.816 77.4844L-65.861 77.4844" stroke="url(#paint3_linear_6858_42709)" />
    </g>
    <defs>
      <linearGradient
        id="paint0_linear_6858_42709"
        x1="7.08966"
        y1="-16.7183"
        x2="7.08965"
        y2="149.48"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#FEFEFE" />
        <stop offset="0.535" stopColor="#9ADAB3" />
        <stop offset="1" stopColor="#FEFEFE" />
      </linearGradient>
      <linearGradient
        id="paint1_linear_6858_42709"
        x1="78.238"
        y1="-62.5503"
        x2="78.238"
        y2="108.817"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#FEFEFE" />
        <stop offset="0.535" stopColor="#9ADAB3" />
        <stop offset="1" stopColor="#FEFEFE" />
      </linearGradient>
      <linearGradient
        id="paint2_linear_6858_42709"
        x1="178.247"
        y1="6.27734"
        x2="-12.2946"
        y2="6.27734"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#FEFEFE" />
        <stop offset="0.535" stopColor="#9ADAB3" />
        <stop offset="1" stopColor="#FEFEFE" />
      </linearGradient>
      <linearGradient
        id="paint3_linear_6858_42709"
        x1="105.816"
        y1="77.9844"
        x2="-65.861"
        y2="77.9844"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#FEFEFE" />
        <stop offset="0.535" stopColor="#9ADAB3" />
        <stop offset="1" stopColor="#FEFEFE" />
      </linearGradient>
      <clipPath id="clip0_6858_42709">
        <rect width="84" height="84" fill="white" />
      </clipPath>
    </defs>
  </svg>
);
