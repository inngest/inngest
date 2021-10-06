import React from "react";
import styled from "@emotion/styled";

export const Tabs = styled.div`
  display: flex;
  flex-direction: row;
  align-items: stretch;
  justify-content: flex-start;
  border-bottom: 1px solid #efeeea;
  font-size: 16px;
  position: relative;
  z-index: 2;

  > a {
    padding: 8px 0;
    display: flex;
    align-items: center;
    color: inherit;
    text-decoration: none;
    opacity: 0.7;
    transition: all 0.3s;

    color: #2f6d9d;

    &:hover {
      opacity: 1;
    }
  }
  a + a {
    margin-left: 30px;
  }

  a.active {
    opacity: 1;
    font-weight: 500;
    box-shadow: inset 0 -2px 0 #2f6d9d;
  }
`;

type TabProps = {
  to?: string;
  active?: boolean;
  onClick?: () => void;
};

export const Tab: React.FC<TabProps> = ({ to, onClick, active, children }) => {
  return (
    <a onClick={onClick} className={active ? "active" : ""}>
      {children}
    </a>
  );
};
