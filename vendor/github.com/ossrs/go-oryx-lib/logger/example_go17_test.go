// The MIT License (MIT)
//
// Copyright (c) 2013-2017 Oryx(ossrs)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// +build go1.7

package logger_test

import (
	"context"
	ol "github.com/ossrs/go-oryx-lib/logger"
)

func ExampleLogger_ContextGO17() {
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Wrap the context, to support CID.
	ctx = ol.WithContext(ctx)

	// We must wrap the context.
	// For each coroutine or request, we must use a context.
	go func(ctx context.Context) {
		ol.T(ctx, "Log with context")

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		func(ctx context.Context) {
			ol.T(ctx, "Log in child function")
		}(ctx)
	}(ctx)
}

func ExampleLogger_MultipleContextGO17() {
	ctx := context.Background()

	pfn := func(ctx context.Context) {
		ol.T(ctx, "Log with context")

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		func(ctx context.Context) {
			ol.T(ctx, "Log in child function")
		}(ctx)
	}

	// We must wrap the context.
	// For each coroutine or request, we must use a context.
	func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Wrap the context, to support CID.
		ctx = ol.WithContext(ctx)
		go pfn(ctx)
	}(ctx)

	// Another goroutine, use another context if they aren't in the same scope.
	func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Wrap the context, to support CID.
		ctx = ol.WithContext(ctx)
		go pfn(ctx)
	}(ctx)
}

func ExampleLogger_AliasContext() {
	// This is the source context.
	source := ol.WithContext(context.Background())

	// We should inherit from the parent context.
	parent := context.Background()

	// However, we maybe need to create a context from parent,
	// but with the same cid of source.
	ctx := ol.AliasContext(parent, source)

	// Now use the context, which has the same cid of source,
	// and it belongs to the parent context tree.
	_ = ctx
}
