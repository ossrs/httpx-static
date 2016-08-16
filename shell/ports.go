/*
The MIT License (MIT)

Copyright (c) 2016 winlin

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

/*
 The ports manager for shell.
*/
package main

import "fmt"

// The port pool manage available ports.
type PortPool struct {
	shell *ShellBoss
	// alloc new port from ports, fill ports from left,
	// release to ports when free port.
	ports []int
	left  []int
}

// alloc port in [start,stop]
func NewPortPool(start, stop int) *PortPool {
	v := &PortPool{}
	for i := start; i <= stop; i++ {
		if len(v.ports) < 64 {
			v.ports = append(v.ports, i)
		} else {
			v.left = append(v.left, i)
		}
	}
	return v
}

func (v *PortPool) Alloc(nbPort int) (ports []int, err error) {
	if nbPort <= 0 {
		return nil, fmt.Errorf("invalid ports %v", nbPort)
	}
	if len(v.ports)+len(v.left) < nbPort {
		return nil, fmt.Errorf("no %v port available, left %v", nbPort, len(v.ports)+len(v.left))
	}

	if len(v.ports) < nbPort {
		cp := nbPort - len(v.ports)
		v.ports = append(v.ports, v.left[0:cp]...)
		v.left = v.left[cp:]
	}

	ports = v.ports[0:nbPort]
	v.ports = v.ports[nbPort:]
	return
}

func (v *PortPool) Free(port int) {
	v.ports = append(v.ports, port)
}
