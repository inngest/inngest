import IconTheme from "./theme";

export function IconBlog({ size = 20, className = "", color = "transparent" }) {
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
          x="-32.2%"
          y="-23%"
          width="164.4%"
          height="164.4%"
          filterUnits="objectBoundingBox"
          id="blog-a"
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
          d="m14.3658 10.6931-3.6662 3.6687a1.0237 1.0237 0 0 0-.2598-.1011l-4.1064-1.0255a2.176 2.176 0 0 1-1.6166-1.7477l-1.198-7.1567A.297.297 0 0 1 3.5 4.2542l4.0242 4.0269a.9513.9513 0 0 1 .267-.4839l.0072-.0072a.9597.9597 0 0 1 .4908-.2744L4.2765 3.5003a.1998.1998 0 0 1 .065.0072l7.152 1.1988A2.1758 2.1758 0 0 1 13.24 6.324l1.0248 4.1092c.0217.0908.0557.1783.101.26Z"
          id="blog-b"
        />
      </defs>
      <g fill="none" fillRule="nonzero">
        <use fill="#000" filter="url(#blog-a)" xlinkHref="#blog-b" />
        <use fill="#FFF" xlinkHref="#blog-b" />
        <path
          d="M16.2927 13.6974 13.702 16.29a.7214.7214 0 0 1-1.0176 0l-1.7465-1.7477a1.1505 1.1505 0 0 0-.2382-.1805l3.6662-3.6687c.0495.087.1102.1671.1804.2384l1.7465 1.7476a.7225.7225 0 0 1 0 1.0183ZM9.3306 9.321c-.4234.4122-1.0988.4086-1.5178-.008-.419-.4165-.427-1.0923-.018-1.5187l.005-.0057c.423-.423 1.1084-.4229 1.5311.0003.4227.4232.4226 1.1091-.0003 1.5321Z"
          fill={theme.color}
          opacity={theme.opacity}
        />
      </g>
    </svg>
  );
}
