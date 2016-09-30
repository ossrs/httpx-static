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

import (
	"fmt"
	"net"
	"strconv"
)

// PortPool The port pool manage available ports.
type PortPool struct {
	// Alloc new port from ports. Alloc new port from the head of the list.
	// Release to ports when free port.
	// If check failed,  put the port to the tail of the list
	ports []int
	// The ports in use. fill used when alloc
	used []int
}

// NewPortPool alloc port in [start,stop]
func NewPortPool(start, stop int) *PortPool {
	v := &PortPool{}
	v.ports = make([]int, stop-start+1)
	for i := start; i <= stop; i++ {
		v.ports[i-start] = i
	}
	return v
}

// allocOnePort alloc a port from the port pool
func (v *PortPool) allocOnePort() (int, error) {
	if len(v.ports) <= 0 {
		return 0, fmt.Errorf("empty port pool")
	}

	for i, p := range v.ports {
		if checkPort(p) {
			// delete i-index element
			v.ports = append(v.ports[:i], v.ports[i+1:]...)
			v.used = append(v.used, p)
			return p, nil
		}
	}

	return 0, fmt.Errorf("No available port")
}

// Alloc alloc nbPort ports from the port pool
func (v *PortPool) Alloc(nbPort int) ([]int, error) {
	if nbPort <= 0 {
		return nil, fmt.Errorf("invalid ports %v", nbPort)
	}

	if len(v.ports) < nbPort {
		return nil, fmt.Errorf("no %v port available, left %v", nbPort, len(v.ports))
	}

	ports := []int{}
	for i := 0; i < nbPort; i++ {
		p, err := v.allocOnePort()
		if err != nil {
			return nil, err
		} else {
			ports = append(ports, p)
		}
	}
	return ports, nil
}

// Free free port from the port pool
func (v *PortPool) Free(port int) {
	// Free used ports to the head of the ports list for better port reused.
	v.ports = append([]int{port}, v.ports...)
	for i, p := range v.used {
		if port == p {
			// delete i-index element
			v.used = append(v.used[:i], v.used[i+1:]...)
			return
		}
	}
}

// GetPortsInUse get a list of port in use
func (v *PortPool) GetPortsInUse() []int {
	return v.used
}

// checkPort check if a port is available
func checkPort(port int) bool {
	// Concatenate a colon and the port
	host := ":" + strconv.Itoa(port)

	// Try to create a server with the port
	listener, err := net.Listen("tcp", host)

	// if it fails then the port is likely taken
	if err != nil {
		return false
	}

	// close the server
	listener.Close()

	// we successfully used and closed the port
	// so it's now available to be used again
	return true
}
