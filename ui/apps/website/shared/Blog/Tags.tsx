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
        <span
          className="bg-indigo-500/30 text-indigo-200 text-sm inline-flex px-2.5 py-1 rounded"
          key={t}
        >
          {TagMap[t]}
        </span>
      ))}
    </TagContainer>
  );
};

const TagContainer = styled.span`
  display: inline-block;
  margin-left: 0.5rem; ;
`;

export default Tags;
