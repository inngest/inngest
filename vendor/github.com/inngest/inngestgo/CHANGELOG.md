# inngestgo

## 0.14.0

### Minor Changes

- 097ebf2: Add support for steps in HTTP endpoints

## 0.13.1

### Patch Changes

- f122af6: Add parallel mode option. Fix parallel step reporting when not targeted
- 6c3b145: Fix SDK failing to reconnect when gateways are rotated

## 0.13.0

### Minor Changes

- f54d7a8: Add step.WaitForSignal
- 4869295: Rename function options from Fn${Option} to Config${Option}
- f54d7a8: Add step.WaitForSignal

### Patch Changes

- 36a3186: Add support for cancel mode in function singletons

## 0.12.0

### Minor Changes

- 4cf0281: Add support for function singletons
- 9d45eaf: Connect: Reliability improvements
- 7aec433: Update function configuration types to always use inngestgo.Fn imports

### Patch Changes

- 9373b31: Clean up request leases properly
- c68c629: Change LoggerFromContext to not return an error
