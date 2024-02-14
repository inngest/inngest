export default ({
  outline = true,
  fill = "#222631",
  size = "24",
}: {
  outline?: boolean;
  fill?: string;
  size?: string | number;
}) => {
  if (outline === undefined || outline) {
    return (
      <svg
        width={size}
        height={size}
        viewBox="2 2 20 20"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          d="M12 22C6.47715 22 2 17.5228 2 12C2 6.47715 6.47715 2 12 2C17.5228 2 22 6.47715 22 12C21.9939 17.5203 17.5203 21.9939 12 22ZM4 12.172C4.04732 16.5732 7.64111 20.1095 12.0425 20.086C16.444 20.0622 19.9995 16.4875 19.9995 12.086C19.9995 7.68451 16.444 4.10977 12.0425 4.086C7.64111 4.06246 4.04732 7.59876 4 12V12.172ZM10 16.5V7.5L16 12L10 16.5Z"
          fill={fill}
        ></path>
      </svg>
    );
  }

  return (
    <svg
      width={size}
      height={size}
      viewBox="2 2 20 20"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M12 22C6.47967 21.994 2.00606 17.5204 2 12V11.8C2.10993 6.30455 6.63459 1.92797 12.1307 2.0009C17.6268 2.07382 22.0337 6.5689 21.9978 12.0654C21.9619 17.5618 17.4966 21.9989 12 22ZM10 7.50002V16.5L16 12L10 7.50002Z"
        fill={fill}
      ></path>
    </svg>
  );
};
