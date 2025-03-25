import { getQuickJS, shouldInterruptAfterDeadline } from 'quickjs-emscripten';

type Disposable = { readonly alive: boolean; dispose(): void };

class DisposableComposer {
  storage: Array<Disposable> = [];

  add(disposable: Disposable) {
    this.storage.unshift(disposable);
  }

  dispose() {
    this.storage.forEach((disposable) => {
      if (disposable.alive) {
        disposable.dispose();
      } else {
        console.warn('Warning: attempting to dispose already-freed lifetime');
      }
    });
  }
}

/**
 * Create a QuickJS VM that will only live for a short time;
 * because it will only live for a short time, no handles to JSValues
 * will be disposed until you dispose the entire vm. This simplifies the
 * memory management requirements, so you don't need to worry about
 * disposing handles.
 *
 * You can optionally specify a timeout in ms, and if the script doesn't
 * complete within that amount of time, then it will be terminated.
 */
export default async function makeVM(timeout: number = -1) {
  const disposables = new DisposableComposer();

  const QuickJS = await getQuickJS();
  const vm = QuickJS.createVm();
  let alive = true;
  disposables.add({
    get alive() {
      return alive;
    },
    dispose: () => {
      vm.dispose();
      alive = false;
    },
  });

  if (timeout !== -1) {
    vm.setInterruptHandler(shouldInterruptAfterDeadline(Date.now() + timeout));
  }

  const trueHandle = vm.unwrapResult(vm.evalCode('true'));
  const falseHandle = vm.unwrapResult(vm.evalCode('false'));
  const nullHandle = vm.unwrapResult(vm.evalCode('null'));
  const errorHandle = vm.getProp(vm.global, 'Error');
  disposables.add(errorHandle);

  /**
   * Create a JSValue handle from a native JavaScript value,
   * by creating an equivalent value within the QuickJS vm.
   *
   * It works for JSON-serializable values, primitives, and functions.
   * For functions, the arguments and return value may only be
   * JSON-serializable values or primitives.
   */
  function marshal(target: any) {
    switch (typeof target) {
      case 'number': {
        const handle = vm.newNumber(target);
        disposables.add(handle);
        return handle;
      }
      case 'string': {
        const handle = vm.newString(target);
        disposables.add(handle);
        return handle;
      }
      case 'undefined': {
        return vm.undefined;
      }
      case 'boolean': {
        return target ? trueHandle : falseHandle;
      }
      case 'object': {
        if (target === null) {
          return nullHandle;
        }
        if (Array.isArray(target)) {
          const array = vm.newArray();
          disposables.add(array);

          target.forEach((item) => {
            const marshaledItem = marshal(item);

            // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- The type isn't falsy but it's unclear whether it can be falsy at run time.
            if (!marshaledItem) {
              return vm.undefined;
            }

            disposables.add(marshaledItem);
            const push = vm.getProp(array, 'push');
            vm.callFunction(push, array, marshaledItem);
          });

          return array;
        }

        if (target instanceof Promise) {
          const deferred = vm.newPromise();
          disposables.add(deferred);

          target.then((result: any) => {
            const marshaled = marshal(result);
            deferred.resolve(marshaled);
            vm.executePendingJobs(-1);
          });
          return deferred.handle;
        }

        const obj = vm.newObject();
        disposables.add(obj);

        for (let key in target) {
          const value = target[key];

          const marshaledKey = marshal(key);

          const marshaledValue = (() => {
            if (typeof value !== 'function') {
              return marshal(value);
            }

            // If the object contains a function, ensure that we call the function in the
            // scope of the object.
            //
            // This is particularly important for Response() instances;  eg. await res.text()
            // does not work unless we properly scope this call.
            return marshal(function () {
              return target[key].apply(target, arguments);
            });
          })();

          // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- The type isn't falsy but it's unclear whether it can be falsy at run time.
          if (!marshaledKey || !marshaledValue) {
            continue;
          }

          vm.setProp(obj, marshaledKey, marshaledValue);
        }

        return obj;
      }
      case 'function': {
        const handle = vm.newFunction(target.name || '<anonymous function>', (...handles) => {
          const unmarshaledArgs = handles.map((handle) => vm.dump(handle));
          let result = undefined;
          try {
            result = target(unmarshaledArgs);
          } catch (err: any) {
            const errResult = vm.callFunction(errorHandle, vm.undefined, marshal(err.message));
            if (errResult.error) {
              const context = vm.dump(errResult.error);
              throw new Error('Failed to create error: ' + JSON.stringify(context));
            } else {
              // @ts-ignore
              throw errResult.value;
            }
          }
          return marshal(result);
        });
        disposables.add(handle);
        return handle;
      }
      default: {
        throw new Error(`${typeof target} marshaling is not supported`);
      }
    }
  }

  const newNumber: typeof vm.newNumber = (num) => {
    const handle = vm.newNumber(num);
    disposables.add(handle);
    return handle;
  };

  const newString: typeof vm.newString = (str) => {
    const handle = vm.newString(str);
    disposables.add(handle);
    return handle;
  };

  const newObject: typeof vm.newObject = (prototype) => {
    const handle = vm.newObject(prototype);
    disposables.add(handle);
    return handle;
  };

  const newArray: typeof vm.newArray = () => {
    const handle = vm.newArray();
    disposables.add(handle);
    return handle;
  };

  const newFunction: typeof vm.newFunction = (name, fn) => {
    const handle = vm.newFunction(name, fn);
    disposables.add(handle);
    return handle;
  };

  function typedBind<T extends Function>(func: T, target: any): T {
    return func.bind(target);
  }

  // Create a new 'log' function for debugging.
  const log = marshal(function () {
    Array.from(arguments).forEach((a) => {
      console.log(a[0]);
    });
  });
  const vmConsole = newObject();
  vm.setProp(vmConsole, 'log', log);
  vm.setProp(vm.global, 'console', vmConsole);

  // Add fetch, but ensure we never include credentials.
  // Create a new 'fetch' function, which _always_ fetches without credentials.
  vm.setProp(
    vm.global,
    'fetch',
    newFunction('fetch', function (url: any, options: any) {
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- The type isn't falsy but it's unclear whether it can be falsy at run time.
      if (!vm) {
        return;
      }
      const opts = vm.dump(options) || {};
      const result = fetch(
        vm.getString(url),
        // NEVER include credentials, else the eval()ed JS may be able to request
        // any data from GQL as the current user.
        Object.assign({}, opts, { credentials: 'omit' })
      );
      return marshal(result);
    })
  );

  return {
    /**
     * [`undefined`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/undefined).
     */
    undefined: vm.undefined,

    /**
     * [`global`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects).
     * A handle to the global object inside the interpreter.
     * You can set properties to create global variables.
     */
    global: vm.global,

    /**
     * `typeof` operator. **Not** [standards compliant](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/typeof).
     *
     * @remarks
     * Does not support BigInt values correctly.
     */
    typeof: typedBind(vm.typeof, vm),

    /**
     * Converts a Javascript number into a QuickJS value.
     */
    newNumber,

    /**
     * Converts `handle` into a Javascript number.
     * @returns `NaN` on error, otherwise a `number`.
     */
    getNumber: typedBind(vm.getNumber, vm),

    /**
     * Create a QuickJS [string](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String) value.
     */
    newString,

    newArray,

    /**
     * Converts `handle` to a Javascript string.
     */
    getString: typedBind(vm.getString, vm),

    /**
     * `{}`.
     * Create a new QuickJS [object](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Object_initializer).
     *
     * @param prototype - Like [`Object.create`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Object/create).
     */
    newObject,

    /**
     * Convert a Javascript function into a QuickJS function value.
     * See [[VmFunctionImplementation]] for more details.
     *
     * A [[VmFunctionImplementation]] should not free its arguments or its retun
     * value. A VmFunctionImplementation should also not retain any references to
     * its veturn value.
     */
    newFunction,

    /**
     * `handle[key]`.
     * Get a property from a JSValue.
     *
     * @param key - The property may be specified as a JSValue handle, or as a
     * Javascript string (which will be converted automatically).
     */
    getProp: typedBind(vm.getProp, vm),

    /**
     * `handle[key] = value`.
     * Set a property on a JSValue.
     *
     * @remarks
     * Note that the QuickJS authors recommend using [[defineProp]] to define new
     * properties.
     *
     * @param key - The property may be specified as a JSValue handle, or as a
     * Javascript string (which will be converted automatically).
     */
    setProp: typedBind(vm.setProp, vm),

    /**
     * [`Object.defineProperty(handle, key, descriptor)`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Object/defineProperty).
     *
     * @param key - The property may be specified as a JSValue handle, or as a
     * Javascript string (which will be converted automatically).
     */
    defineProp: typedBind(vm.defineProp, vm),

    /**
     * [`func.call(thisVal, ...args)`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Function/call).
     * Call a JSValue as a function.
     *
     * See [[unwrapResult]], which will throw if the function returned an error, or
     * return the result handle directly.
     *
     * @returns A result. If the function threw, result `error` is a handle to the exception.
     */
    callFunction: typedBind(vm.callFunction, vm),

    /**
     * Like [`eval(code)`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/eval#Description).
     * Evaluates the Javascript source `code` in the global scope of this VM.
     *
     * See [[unwrapResult]], which will throw if the function returned an error, or
     * return the result handle directly.
     *
     * *Note*: to protect against infinite loops, provide an interrupt handler to
     * [[setInterruptHandler]]. You can use [[InterruptAfterDeadline]] to
     * create a time-based deadline.
     *
     * @returns The last statement's value. If the code threw, result `error` will be
     * a handle to the exception. If execution was interrupted, the error will
     * have name `InternalError` and message `interrupted`.
     */
    evalCode: (code: string) => {
      const result = vm.evalCode(code);
      // @ts-ignore
      if (result.value) {
        disposables.add(
          // @ts-ignore
          result.value
        );
      } else {
        result.error && disposables.add(result.error);
      }
      return result;
    },

    /**
     * Dump a JSValue to Javascript in a best-effort fashion.
     * Returns `handle.toString()` if it cannot be serialized to JSON.
     */
    dump: typedBind(vm.dump, vm),

    executePendingJobs: typedBind(vm.executePendingJobs, vm),

    /**
     * Unwrap a VmCallResult, returning it's value on success, and throwing the dumped
     * error on failure.
     */
    unwrapResult: typedBind(vm.unwrapResult, vm),

    /**
     * Set a callback which is regularly called by the QuickJS engine when it is
     * executing code. This callback can be used to implement an execution
     * timeout.
     *
     * The interrupt handler can be removed with [[removeInterruptHandler]].
     */
    setInterruptHandler: typedBind(vm.setInterruptHandler, vm),

    /**
     * Remove the interrupt handler, if any.
     * See [[setInterruptHandler]].
     */
    removeInterruptHandler: typedBind(vm.removeInterruptHandler, vm),

    /**
     * Dispose of this VM's underlying resources.
     */
    dispose: typedBind(disposables.dispose, disposables),

    marshal,
  };
}
