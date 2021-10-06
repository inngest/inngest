import { InputEditor, Props } from "./InputEditor";

export default {
  component: InputEditor,
  title: "Components/InputEditor",
};

const Editor = (props: Props) => <InputEditor {...props} />;

export const Input = Editor.bind({});
// @ts-ignore
Input.args = {
  kind: "input",
  value: "hi {{ there | upper }}!",
};

export const Textarea = Editor.bind({});
// @ts-ignore
Textarea.args = {
  kind: "textarea",
};

export const Code = Editor.bind({});
// @ts-ignore
Code.args = {
  kind: "code",
};

export const JS = Editor.bind({});
// @ts-ignore
JS.args = {
  kind: "code",
  language: "javascript",
};

export const Multiple = () => (
  <div>
    <label>
      Name
      <InputEditor kind="input" />
    </label>
    <br />
    <label style={{ marginTop: 20 }}>
      Email
      <InputEditor kind="input" />
    </label>
  </div>
);
