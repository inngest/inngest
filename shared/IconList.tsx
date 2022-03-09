import React from "react";
import styled from "@emotion/styled";

type IconListProps = {
  direction?: "horizontal" | "vertical";
  collapseWidth?: number | string;
  circles?: boolean;
  size?: "small" | "default";
  items: Array<{
    icon: React.FC<any>;
    text: string | React.ReactFragment;
    quantity?: string;
  }>;
};

const IconList: React.FC<IconListProps> = ({
  direction = "horizontal",
  collapseWidth,
  circles = true,
  size = "default",
  items = [],
}) => {
  return (
    <List direction={direction} collapseWidth={collapseWidth} size={size}>
      {items.map((item, idx) => (
        <ListItem key={idx}>
          <IconWrapper circle={circles}>
            <item.icon size="1em" color="#fff" />
          </IconWrapper>
          <span>
            {item.quantity ? (
              <>
                <strong>{item.quantity}</strong> {item.text}
              </>
            ) : (
              item.text
            )}
          </span>
        </ListItem>
      ))}
    </List>
  );
};

const List = styled.ul<{
  direction: "horizontal" | "vertical";
  collapseWidth: number | string;
  size: "small" | "default";
}>`
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: ${(props) =>
    props.direction === "vertical" ? "column" : "row"};
  font-family: var(--font-mono);
  font-size: ${(props) => (props.size === "small" ? "0.7em" : "inherit")};

  li + li {
    margin: ${(props) =>
      props.direction === "vertical" ? "1em 0 0" : "0 0 0 3em"};
  }

  // Collapse the list at the given screen width
  @media (max-width: ${(props) => props.collapseWidth || "0"}px) {
    flex-direction: column;
    li {
      margin-left: 0 !important;
    }
  }
`;

const ListItem = styled.li`
  display: flex;
  align-items: center;
  height: 2rem;
  line-height: 1.2rem;
  margin: 0;
  padding: 0;
`;

const IconWrapper = styled.div<{ circle: boolean }>`
  display: flex;
  justify-content: center;
  align-items: center;
  flex-shrink: 0;
  height: 1.6em;
  width: 1.6em;
  margin-right: 0.5em;
  background: ${(props) => (props.circle ? "var(--primary-color)" : "none")};
  border-radius: 50%;
`;

export default IconList;
