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
      <g fill="none" fill-rule="evenodd">
        <path d="M0 0h20v20H0z" />
        <path
          d="M4.1667 2.5H5a.8333.8333 0 0 1 .8333.8333v13.3334A.8333.8333 0 0 1 5 17.5h-.8333a.8333.8333 0 0 1-.8334-.8333V3.3333A.8333.8333 0 0 1 4.1667 2.5Zm4.1666 0h.8334A.8333.8333 0 0 1 10 3.3333v13.3334a.8333.8333 0 0 1-.8333.8333h-.8334a.8333.8333 0 0 1-.8333-.8333V3.3333A.8333.8333 0 0 1 8.3333 2.5Z"
          fill={theme.color}
          opacity={theme.opacity}
          fill-rule="nonzero"
        />
        <path
          d="m12.1765 2.9446.4728-.1628c.5222-.1798 1.0912.0978 1.271.62l4.2324 12.2917c.1798.5222-.0977 1.0913-.62 1.2711l-.4727.1628c-.5222.1798-1.0913-.0978-1.271-.62L11.5564 4.2157c-.1798-.5222.0978-1.0913.62-1.2711Z"
          fill="#FFF"
          fill-rule="nonzero"
        />
      </g>
    </svg>
  );
}
