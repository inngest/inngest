import styled from "@emotion/styled";
import React, { useState } from "react";
import CodeWindow from "src/shared/CodeWindow";

const defaultCode = `
  import { createFunction } from "inngest"

  createFunction("Signup", "app/user.signup", async ({ event }) => {
    // ...
  })
`;

export default function CodeSnippet() {
  const [code, setCode] = useState<string>(defaultCode);
  const [filename, setFilename] = useState<string>("");
  const [backgroundColor, setBackgroundColor] = useState<string>("#ffffff");
  const [borderColor, setBorderColor] = useState<string>("#94a3b8");
  const [zoom, setZoom] = useState<string>("160");
  return (
    <Container>
      <div className="py-36">
        <CodeWindow
          snippet={code}
          filename={filename}
          className="inline-block my-8 pr-8 border-2"
          style={{
            backgroundColor,
            borderColor,
            transform: `scale(${zoom}%)`,
          }}
        />
      </div>
      <div className="mt-16 w-full text-xs">
        <h2>Controls</h2>
        <div className="my-2">
          <label>
            Color{" "}
            <input
              type="color"
              name="color"
              defaultValue={backgroundColor}
              className="p-0"
            />
          </label>
        </div>
        <div className="my-2">
          <label>
            Border Color{" "}
            <input
              type="color"
              name="borderColor"
              defaultValue={borderColor}
              onChange={(e) => {
                setBorderColor(e.target.value);
              }}
              className="p-0"
            />
          </label>
        </div>
        <div className="my-2">
          <label>
            Zoom{" "}
            <input
              type="number"
              className="text-xs"
              style={{ border: "1px solid #94a3b8" }}
              onChange={(e) => setZoom(e.target.value)}
              defaultValue={zoom}
            />
          </label>
        </div>
        <div className="my-2">
          <label>
            Filename{" "}
            <input
              type="text"
              className="text-xs"
              style={{ border: "1px solid #94a3b8" }}
              onChange={(e) => setFilename(e.target.value)}
              defaultValue={filename}
            />
          </label>
        </div>
        <div className="my-2">
          <label>
            Code{" "}
            <textarea
              className="w-full h-64 text-xs"
              style={{ border: "1px solid #94a3b8" }}
              onChange={(e) => setCode(e.target.value)}
            >
              {code}
            </textarea>
          </label>
        </div>
      </div>
    </Container>
  );
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  width: 100%;
  padding: 2rem 1rem;
  background: #fff;
`;
