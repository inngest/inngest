import React, { useCallback, useState } from "react";
import debounce from "lodash.debounce";
import styled from "@emotion/styled";
import CodeEditor from "src/shared/CodeEditor";
import { StateProps } from "./state";
import { SerializedStyles } from "@emotion/utils";

type Props = StateProps & {
  wrapperCss?: SerializedStyles;
};

// CueEditor represents the code editor for our config
const CueEditor: React.FC<Props> = ({ dispatch, state, wrapperCss }) => {
  const updateDebounced = useCallback(
    debounce((config: string) => dispatch({ type: "config", config }), 250),
    []
  );

  const [val, setVal] = useState(state.config || "");

  const onUpdate = (val: string) => {
    setVal(val);
    updateDebounced(val);
  };

  return (
    <Wrapper css={wrapperCss}>
      <CodeEditor value={val} onChange={onUpdate} />
    </Wrapper>
  );
};

const Wrapper = styled.div`
  textarea {
    border-top: none;
  }
`;

export default CueEditor;
