import React, { useMemo } from "react";
import styled from "@emotion/styled";
import { useDebounce } from "src/utils/debounce";
import ReactMarkdown from "react-markdown";
import {
  WorkflowMetadata,
  WorkflowFormSelect,
  WorkflowFormInput,
  WorkflowFormTextarea,
  WorkflowFormDatetime,
  IntegrationEvent,
} from "src/types";
import { InputEditor } from "src/shared/InputEditor/InputEditor";
import { Template } from "@tonyhb/rs-templating";
import { P as DefaultProps } from "../consts";
import { useWorkflowContext } from "../../state";
import { showAvailableActionData } from "../../data";
import { CompletionItemSubset } from "src/types";
import { monaco } from "react-monaco-editor";

type P = Omit<
  DefaultProps,
  | "onChange"
  | "workspaceID"
  | "abstractAction"
  | "onMetadataKeyChange"
  | "setDirty"
>;

export type Overrides = {
  overrides?: {
    [name: string]: React.FC<FormFieldProps>;
  };

  // title overrides the workflowMetadata title, if provided.
  // this is automatically configured and should be left blank for most inputs.
  title?: string;
  // value overrides the value derived from the state metadata, if provided.
  // this is automatically configured and should be left blank for most inputs.
  value?: any;

  style?: any;
};

// useValue is a hook which returns the form field value given the form field props.
// This templates the form field if previewing templates is on.
export const useValue = function (
  props: P & { wm: WorkflowMetadata },
  explicitValue?: any
) {
  const { state } = props;
  const wm = props.wm || {};

  const value =
    explicitValue === undefined ? state.metadata[wm.name] : explicitValue;

  const variables = useVariables(value);
  const data = React.useMemo(() => {
    if (!props.previewTemplates || !state.availableData) {
      return {};
    }
    return state.availableData.displayJSON;
  }, [state.availableData, props.previewTemplates]);

  if (!wm || !wm.form) return [null, {}];

  let newVal = value;

  if (props.previewTemplates) {
    try {
      newVal = Template.compile_and_execute(
        value || "",
        JSON.stringify(data) || "{}"
      );
    } catch (e) {}
  }

  return [newVal, variables];
};

export type FormFieldProps = P &
  Overrides & {
    wm: WorkflowMetadata;
    onChange: (field: string, value: any) => void;
    value?: any;
  };

export const FormField: React.FC<FormFieldProps> = (props) => {
  const { overrides } = props;
  const wm = props.wm || {};

  const C = overrides && overrides[wm.name];
  const [value, variables] = useValue(props, props.value);

  if (!wm || !wm.form) return null;

  if (C) {
    return <C {...props} />;
  }

  const field = (() => {
    switch (wm.form.type) {
      case "select":
        return <FormSelect {...props} form={wm.form} value={value} />;
      case "input":
        return <FormInput {...props} form={wm.form} value={value} />;
      case "datetime":
        return <FormInput {...props} form={wm.form} value={value} />;
      case "textarea":
        return <FormTextarea {...props} form={wm.form} value={value} />;
      case "toggle":
      default:
        return null;
    }
  })();

  return (
    <Wrapper style={props.style}>
      <label>
        {props.title !== undefined ? props.title : wm.form.title}
        {getHint(wm.form.hint)}
        {field}

        {/*variables.data.length > 0 && (
          <Vars>
            <p>This field contains the following variables:</p>
            <span>
              {variables.data.map((v: string) => (
                <code key={v}>{v}</code>
              ))}
            </span>
          </Vars>
  )*/}

        {variables.error !== null && (
          <Vars>
            <span style={{ color: "red" }}>
              This template is invalid ({variables.error})
            </span>
          </Vars>
        )}
      </label>
    </Wrapper>
  );
};

export const getHint = (hint?: string) => {
  if (!hint) {
    return null;
  }

  return (
    <small>
      <ReactMarkdown
        children={hint}
        linkTarget="_blank"
        components={{
          p: ({ children }) => <span>{children}</span>,
        }}
      />
    </small>
  );
  // We transform links into specific elements, and we transform backticks into
  // code tags.
};

const FormSelect: React.FC<FormFieldProps & { form: WorkflowFormSelect }> = (
  props
) => {
  const { wm, form } = props;

  const onChange = (e: React.SyntheticEvent<HTMLSelectElement>) => {
    if (props.previewTemplates) return;

    if (wm.type === "int") {
      props.onChange(wm.name, parseInt((e.target as HTMLSelectElement).value));
      return;
    }
    props.onChange(wm.name, (e.target as HTMLSelectElement).value);
  };

  return (
    <select
      key={props.previewTemplates ? "preview" : "edit"}
      onChange={onChange}
      value={props.previewTemplates ? props.value : undefined}
      defaultValue={props.value}
      disabled={props.previewTemplates}
      style={{ marginTop: 0 }}
    >
      <option>-</option>
      {form.formselect.choices.map((c) => (
        <option
          key={c.value}
          value={c.value}
          selected={c.value === props.value}
        >
          {c.name}
        </option>
      ))}
    </select>
  );
};

const FormTextarea: React.FC<
  FormFieldProps & { form: WorkflowFormTextarea }
> = (props) => {
  const { wm } = props;

  const onChange = (value: string) => {
    if (props.previewTemplates) return;
    props.onChange(wm.name, value);
  };

  const debounced = useDebounce(onChange);
  const recs = useRecommendations(
    props?.state?.availableData?.displayJSON || {}
  );

  return (
    <>
      <InputEditor
        kind="textarea"
        key={props.previewTemplates ? "preview" : "edit"}
        placeholder={wm.form.placeholder}
        onChange={debounced}
        value={props.value}
        disabled={props.previewTemplates}
        recommendations={recs}
      />
    </>
  );
};

export const FormInput: React.FC<
  FormFieldProps & { form: WorkflowFormInput | WorkflowFormDatetime }
> = (props) => {
  const { wm } = props;

  const onChange = (val: string) => {
    props.onChange(wm.name, val);
  };

  const debounced = useDebounce(onChange);
  const recs = useRecommendations(
    props?.state?.availableData?.displayJSON || {}
  );

  return (
    <InputEditor
      key={props.previewTemplates ? "preview" : "edit"}
      placeholder={wm.form.placeholder}
      onChange={debounced}
      value={props.value}
      disabled={props.previewTemplates}
      recommendations={recs}
    />
  );
};

const useVariables = (value: any) => {
  return useMemo(() => {
    if (!value || typeof value !== "string" || value.indexOf("{") === -1) {
      return { data: [], error: null };
    }
    try {
      return {
        data: Template.new(value).variables,
        error: null,
      };
    } catch {
      return { data: [], error: "Parse error" };
    }
  }, [value]);
};

// Given an object of data from availableData, this produces an array of event recommendations.
const useRecommendations = (data: { [key: string]: any }) => {
  // grab the integration event.
  const [state] = useWorkflowContext();

  return useMemo(() => {
    const fields = state?.integrationEvent?.fields || {};
    const results: Array<string | CompletionItemSubset> = Object.values(
      fields
    ).map((f) => {
      return {
        label: `${f.title} (${f.field})`,
        kind: monaco.languages.CompletionItemKind.Field,
        detail: f.description,
        insertText: f.field,
        documentation: f.description,
      };
    });

    const stack = [{ path: "", data }];
    while (stack.length > 0) {
      const item = stack.pop();
      if (!item) break;

      try {
        Object.keys(item.data).forEach((key) => {
          // Get the value from the object.  If the value is an object, this contains
          // nested data.
          const value = item.data[key];
          const path = item.path === "" ? key : item.path + "." + key;

          if (
            ["string", "number", "boolean", "undefined"].includes(
              typeof value
            ) ||
            value instanceof Date
          ) {
            results.push(path);
            return;
          }

          stack.push({
            path,
            data: value,
          });
        });
      } catch (e) {}
    }
    return results;
  }, [JSON.stringify(data), state.integrationEvent]);
};

const Wrapper = styled.div`
  margin-top: 30px;

  label + & {
  }
`;

const Vars = styled.div`
  margin: 20px 0 30px;
  font-size: 12px;

  p {
    opacity: 0.6;
    margin-bottom: 8px;
  }

  code {
    display: inline-block;
    margin: 0 10px 10px 0;
    background: #d7dfd599;
    padding: 4px 6px;
    border-radius: 4px;
    font-size: 12px;
  }
`;

export default FormField;
