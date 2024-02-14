import React from 'react';
import styled from '@emotion/styled';

const TagMap = {
  'release-notes': 'Release Notes',
  'new-feature': 'New Feature',
};

const Tags: React.FC<{ tags: string[] }> = ({ tags = [] }) => {
  return (
    <TagContainer>
      {tags.map((t) => (
        <span
          className="inline-flex rounded bg-indigo-500/30 px-2.5 py-1 text-sm text-indigo-200"
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
  margin-left: 0.5rem;
`;

export default Tags;
