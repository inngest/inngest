// Copyright (c) 2025 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package dig

// BeforeCallbackInfo contains information about a provided function or decorator
// called by Dig, and is passed to a [BeforeCallback] registered with
// [WithProviderBeforeCallback] or [WithDecoratorBeforeCallback].
type BeforeCallbackInfo struct {
	// Name is the name of the function in the format:
	// <package_name>.<function_name>
	Name string
}

// BeforeCallback is a function that can be registered with a provided function
// using [WithProviderBeforeCallback] or decorator using [WithDecoratorBeforeCallback]
// to cause it to be called before the provided function or decorator is run.
type BeforeCallback func(bci BeforeCallbackInfo)

// WithProviderBeforeCallback returns a [ProvideOption] which has Dig call
// the passed in [BeforeCallback] before the corresponding constructor begins running.
//
// For example, the following prints a message
// before "myConstructor" is called:
//
//	c := dig.New()
//	myCallback := func(bci BeforeCallbackInfo) {
//		fmt.Printf("%q started", bci.Name)
//	}
//	c.Provide(myConstructor, WithProviderBeforeCallback(myCallback)),
//
// BeforeCallbacks can also be specified for Decorators with [WithDecoratorBeforeCallback].
//
// See [BeforeCallbackInfo] for more info on the information passed to the [BeforeCallback].
func WithProviderBeforeCallback(callback BeforeCallback) ProvideOption {
	return withBeforeCallbackOption{
		callback: callback,
	}
}

// WithDecoratorBeforeCallback returns a [DecorateOption] which has Dig call
// the passed in [BeforeCallback] before the corresponding decorator begins running.
//
// For example, the following prints a message
// before "myDecorator" is called:
//
//	c := dig.New()
//	myCallback := func(bci BeforeCallbackInfo) {
//		fmt.Printf("%q started", bci.Name)
//	}
//	c.Decorate(myDecorator, WithDecoratorBeforeCallback(myCallback)),
//
// BeforeCallbacks can also be specified for Constructors with [WithProviderBeforeCallback].
//
// See [BeforeCallbackInfo] for more info on the information passed to the [BeforeCallback].
func WithDecoratorBeforeCallback(callback BeforeCallback) DecorateOption {
	return withBeforeCallbackOption{
		callback: callback,
	}
}

type withBeforeCallbackOption struct {
	callback BeforeCallback
}

var (
	_ ProvideOption  = withBeforeCallbackOption{}
	_ DecorateOption = withBeforeCallbackOption{}
)

func (o withBeforeCallbackOption) applyProvideOption(po *provideOptions) {
	po.BeforeCallback = o.callback
}

func (o withBeforeCallbackOption) apply(do *decorateOptions) {
	do.BeforeCallback = o.callback
}
