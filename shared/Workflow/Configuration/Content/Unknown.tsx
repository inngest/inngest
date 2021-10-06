import React, { useEffect, useState } from "react";
import makeVM from "src/utils/vm";
import { WorkflowMetadata } from "src/types";
import { P as DefaultProps } from "../consts";
import { Overrides, FormField } from "./Field";

// Unknown renders form fields as defined within the action configuration.
const Unknown: React.FC<DefaultProps & Overrides> = (props) => {
  const { abstractAction, state } = props;
  const { WorkflowMetadata: wm } = abstractAction.latest;

  // Set visible fields.
  const [fields, setFields] = useState<WorkflowMetadata[]>([]);

  // compute all fields which are visible, based off of the given answers.
  const compute = async () => {
    const render = await Promise.all(
      wm.map((f) => {
        if (!f.expression) {
          return true;
        }
        return isRenderable(state.metadata, f.expression);
      })
    );

    const renderables = wm.filter((_, index) => !!render[index]);
    setFields(renderables);
  };

  useEffect(() => {
    compute();
  }, [state.metadata]);

  if (!wm || wm.length === 0) {
    return (
      <div>
        <p>This action has no configration</p>
      </div>
    );
  }

  return (
    <div>
      {fields.map((r) => (
        <FormField
          key={r.name}
          wm={r}
          {...props}
          onChange={props.onMetadataKeyChange}
        />
      ))}
    </div>
  );
};

const isRenderable = async (metadata: Object, expression: string) => {
  const vm = await makeVM(Date.now() + 500);
  // Add current action metadata
  vm.setProp(vm.global, "metadata", vm.newObject(vm.marshal(metadata)));

  // execute expression
  const res = vm.evalCode(expression);
  vm.executePendingJobs(-1);
  const unwrapped = vm.unwrapResult(res);
  const ok = vm.dump(unwrapped);

  vm.dispose();
  return !!ok;
};

export default Unknown;
