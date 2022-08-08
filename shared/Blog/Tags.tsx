import React from "react";
import styled from "@emotion/styled";

const TagMap = {
  "release-notes": "Release Notes",
  "new-feature": "New Feature",
};

const Tags: React.FC<{ tags: string[] }> = ({ tags = [] }) => {
  return (
    <TagContainer>
      {tags.map((t) => (
        <Tag key={t}>{TagMap[t]}</Tag>
      ))}
    </TagContainer>
  );
};

const TagContainer = styled.span`
  display: inline-block;
  margin-left: 0.5rem; ;
`;

const Tag = styled.span`
  display: inline-flex;
  font-size: 0.7rem;
  line-height: 1em;
  font-weight: bold;
  color: var(--color-iris-60);
`;
export default Tags;
