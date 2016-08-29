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
 This the main entrance of shell, to start other processes.
*/
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	oh "github.com/ossrs/go-oryx-lib/http"
	ol "github.com/ossrs/go-oryx-lib/logger"
	oo "github.com/ossrs/go-oryx-lib/options"
	"github.com/ossrs/go-oryx/kernel"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var signature = fmt.Sprintf("SHELL/%v", kernel.Version())

const (
	// when active worker failed, restart it.
	workerRestartInterval = time.Duration(5) * time.Second
	// wait for process to start to check api.
	processExecInterval = time.Duration(200) * time.Millisecond
	// max retry to check process.
	processRetryMax = 15
	// api error.
	Success         oh.SystemError = 0
	apiUpgradeError oh.SystemError = 100 + iota
)

// check the api, retry when failed, error when exceed the max.
func checkApi(cmd *exec.Cmd, api string, max int, retry time.Duration) (err error) {
	for i := 0; i < max; i++ {
		// when api ok, success.
		if _, _, err = oh.ApiRequest(api); err == nil {
			return
		}

		// when got state, the process terminated.
		if cmd.ProcessState != nil {
			return
		}

		// retry later.
		time.Sleep(retry)
		continue
	}
	return
}

// retrieve the object from version "major.minor.revision-extra"
func RetrieveVersion(version string) (ver *SrsVersion, err error) {
	ver = &SrsVersion{}

	vers := strings.Split(version, "-")
	if len(vers) > 1 {
		extra := vers[1]
		if ver.Extra, err = strconv.Atoi(extra); err != nil {
			return
		}
	}

	if vers = strings.Split(vers[0], "."); len(vers) != 3 {
		return nil, fmt.Errorf("version invalid syntax")
	}
	if ver.Major, err = strconv.Atoi(vers[0]); err != nil {
		return
	}
	if ver.Minor, err = strconv.Atoi(vers[1]); err != nil {
		return
	}
	if ver.Revision, err = strconv.Atoi(vers[2]); err != nil {
		return
	}

	return
}

// The version of srs.
type SrsVersion struct {
	Major     int    `json:"major"`
	Minor     int    `json:"minor"`
	Revision  int    `json:"revision"`
	Extra     int    `json:"extra"`
	Signature string `json:"signature"`
}

func (v *SrsVersion) String() string {
	if v.Extra <= 0 {
		return fmt.Sprintf("%v.%v.%v", v.Major, v.Minor, v.Revision)
	}
	return fmt.Sprintf("%v.%v.%v-%v", v.Major, v.Minor, v.Revision, v.Extra)
}

// The shell to exec all processes.
type ShellBoss struct {
	conf    *ShellConfig
	rtmplb  *exec.Cmd
	httplb  *exec.Cmd
	apilb   *exec.Cmd
	pool    *kernel.ProcessPool
	ports   *PortPool
	workers []*SrsWorker
	// for upgrade lock.
	activeWorker *SrsWorker
	upgradeLock  *sync.Mutex
	// closed.
	closed    bool
	closeLock *sync.Mutex
}

func NewShellBoss(conf *ShellConfig) *ShellBoss {
	v := &ShellBoss{
		conf:        conf,
		pool:        kernel.NewProcessPool(),
		workers:     make([]*SrsWorker, 0),
		upgradeLock: &sync.Mutex{},
		closeLock:   &sync.Mutex{},
	}

	c := &v.conf.Worker.Ports
	v.ports = NewPortPool(c.Start, c.Stop)
	return v
}

func (v *ShellBoss) Close() (err error) {
	v.closeLock.Lock()
	defer v.closeLock.Unlock()
	if v.closed {
		return
	}
	v.closed = true

	for _, w := range v.workers {
		w.Close()
	}

	return v.pool.Close()
}

func (v *ShellBoss) Upgrade(ctx ol.Context) (err error) {
	v.upgradeLock.Lock()
	defer v.upgradeLock.Unlock()

	if v.activeWorker == nil {
		ol.W(ctx, "update ignore for no active worker")
		return
	}

	var latest *SrsVersion
	if true {
		var b bytes.Buffer
		cmd := exec.Command(v.conf.Worker.Binary, "-v")
		cmd.Stderr = &b
		if err = cmd.Run(); err != nil {
			ol.E(ctx, "upgrade get version failed, err is", err)
			return
		}

		version := strings.TrimSpace(string(b.Bytes()))
		if latest, err = RetrieveVersion(version); err != nil {
			ol.E(ctx, fmt.Sprintf("retrieve version failed, version=%v, err is %v", version, err))
			return
		}
	}

	version := v.activeWorker.version
	if latest.String() == version.String() {
		ol.W(ctx, fmt.Sprintf("upgrade ignore version=%v, current version=%v",
			latest.String(), version.String()))
		return
	}
	ol.T(ctx, fmt.Sprintf("upgrade %v to %v, current signature=%v",
		version.String(), latest.String(), version.Signature))

	// start a new worker.
	// @remark when worker failed, we ignore to close it, for the Cycle() will do this.
	var worker *SrsWorker
	if worker, err = v.execWorker(ctx); err != nil {
		ol.E(ctx, "upgrade exec worker failed, err is", err)
		return
	}

	// upgrade ok, update the active worker and set others to deprecated.
	worker.state = SrsStateActive
	v.activeWorker = worker

	for _, w := range v.workers {
		if worker == w || w.state != SrsStateActive {
			continue
		}

		w.state = SrsStateDeprecated
		r0 := w.cmd.Process.Signal(syscall.SIGUSR2)
		ol.T(ctx, fmt.Sprintf("upgrade notify %v, r0=%v ok", w, r0))
	}

	ol.T(ctx, fmt.Sprintf("upgrade ok, %v", worker))
	return
}

func (v *ShellBoss) ExecBuddies(ctx ol.Context) (err error) {
	// fork processes.
	if r := &v.conf.Rtmplb; r.Enabled {
		args := []string{"-c", r.Config,
			"-a", fmt.Sprintf("tcp://127.0.0.1:%v", r.Api), "-l", fmt.Sprintf("tcp://:%v", r.Rtmp),
		}
		if v.rtmplb, err = v.pool.Start(ctx, r.Binary, args...); err != nil {
			ol.E(ctx, "exec rtmplb failed, err is", err)
			return
		}
		p := v.rtmplb
		ol.T(ctx, fmt.Sprintf("exec rtmplb ok, args=%v, pid=%v", p.Args, p.Process.Pid))
	}

	if r := &v.conf.Httplb; r.Enabled {
		args := []string{"-c", r.Config,
			"-a", fmt.Sprintf("tcp://127.0.0.1:%v", r.Api), "-l", fmt.Sprintf("tcp://:%v", r.Http),
		}
		if v.httplb, err = v.pool.Start(ctx, r.Binary, args...); err != nil {
			ol.E(ctx, "exec httplb failed, err is", err)
			return
		}
		p := v.httplb
		ol.T(ctx, fmt.Sprintf("exec httplb ok, args=%v, pid=%v", p.Args, p.Process.Pid))
	}

	if r := &v.conf.Apilb; r.Enabled {
		args := []string{"-c", r.Config,
			"-api", fmt.Sprintf("tcp://127.0.0.1:%v", r.Api),
			"-backend", fmt.Sprintf("tcp://:%v", r.Backend),
		}
		if v.apilb, err = v.pool.Start(ctx, r.Binary, args...); err != nil {
			ol.E(ctx, "exec apilb failed, err is", err)
			return
		}
		p := v.apilb
		ol.T(ctx, fmt.Sprintf("exec apilb ok, args=%v, pid=%v", p.Args, p.Process.Pid))
	}

	// sleep for a while and check the api.
	api := fmt.Sprintf("http://127.0.0.1:%v/api/v1/version", v.conf.Rtmplb.Api)
	if err = checkApi(v.rtmplb, api, processRetryMax, processExecInterval); err != nil {
		ol.E(ctx, fmt.Sprintf("rtmplb failed, api=%v, max=%v, interval=%v, err is %v",
			api, processRetryMax, processExecInterval, err))
		return
	}
	api = fmt.Sprintf("http://127.0.0.1:%v/api/v1/version", v.conf.Httplb.Api)
	if err = checkApi(v.httplb, api, processRetryMax, processExecInterval); err != nil {
		ol.E(ctx, fmt.Sprintf("httplb failed, api=%v, max=%v, interval=%v, err is %v",
			api, processRetryMax, processExecInterval, err))
		return
	}
	api = fmt.Sprintf("http://127.0.0.1:%v/api/v1/version", v.conf.Apilb.Api)
	if err = checkApi(v.apilb, api, processRetryMax, processExecInterval); err != nil {
		ol.E(ctx, fmt.Sprintf("apilb failed, api=%v, max=%v, interval=%v, err is %v",
			api, processRetryMax, processExecInterval, err))
		return
	}
	ol.T(ctx, "kernel process ok.")

	// fork workers.
	// @reamrk when worker failed, quit.
	var worker *SrsWorker
	if worker, err = v.execWorker(ctx); err != nil {
		ol.E(ctx, "exec worker failed, err is", err)
		return
	}
	v.activeWorker = worker
	worker.state = SrsStateActive
	ol.T(ctx, fmt.Sprintf("worker process ok, %v", worker))

	ol.T(ctx, fmt.Sprintf("all buddies ok"))
	return
}

// Cycle all processes util quit.
func (v *ShellBoss) Cycle(ctx ol.Context) {
	for {
		var err error
		var cmd *exec.Cmd
		if cmd, err = v.pool.Wait(); err != nil {
			// ignore the exit error, which indicates the process terminated not success.
			if _, ok := err.(*exec.ExitError); !ok {
				if err != kernel.PoolDisposed {
					ol.W(ctx, "wait process failed, err is", err)
				}
				return
			} else {
				ol.W(ctx, "process", cmd.Process.Pid, "terminated,", err)
			}
		}

		// ignore events when pool closed
		if v.closed || v.pool.Closed() {
			ol.E(ctx, "pool terminated")
			return
		}

		// when kernel object exited, close pool
		if cmd == v.rtmplb || cmd == v.httplb || cmd == v.apilb {
			ol.E(ctx, fmt.Sprintf("kernel process=%v quit, shell quit.", cmd.Process.Pid))
			v.pool.Close()
			return
		}

		// remove workers
		var worker *SrsWorker
		for i, w := range v.workers {
			if w.cmd == cmd {
				worker = w
				worker.Close()
				v.workers = append(v.workers[:i], v.workers[i+1:]...)
				break
			}
		}

		// ignore the worker not in active.
		if worker.state != SrsStateActive {
			ol.T(ctx, fmt.Sprintf("ignore worker terminated, %v", worker))
			continue
		}
		ol.W(ctx, fmt.Sprintf("restart worker %v", worker))

		// restart worker when terminated.
		for !v.closed {
			if err = v.restartWorker(ctx); err != nil {
				interval := workerRestartInterval
				ol.W(ctx, fmt.Sprintf("ignore and retry %v, err is %v", interval, err))
				time.Sleep(interval)
			} else {
				break
			}
		}
	}

	return
}

func (v *ShellBoss) restartWorker(ctx ol.Context) (err error) {
	v.upgradeLock.Lock()
	defer v.upgradeLock.Unlock()

	var worker *SrsWorker

	v.activeWorker = nil

	if worker, err = v.execWorker(ctx); err != nil {
		// when failed, we must cleanup the worker,
		// because we will retry when failed, and the Cycle()
		// is not reap process when worker is ok.
		worker.Close()

		ol.E(ctx, "restart worker failed, err is", err)
		return
	}
	worker.state = SrsStateActive
	v.activeWorker = worker
	ol.T(ctx, fmt.Sprintf("fork worker ok, %v", worker))

	return
}

func (v *ShellBoss) execWorker(ctx ol.Context) (worker *SrsWorker, err error) {
	r := &v.conf.Worker
	if !r.Enabled {
		return
	}

	if !r.Provider.IsSrs() {
		return
	}

	worker = NewSrsWorker(ctx, v, v.conf.SrsConfig(), v.ports)
	if err = worker.Exec(); err != nil {
		ol.E(ctx, "start srs worker failed, err is", err)
		return
	}
	ol.T(ctx, "start srs worker ok")

	// when worker exec ok, the process must exists,
	// we append the worker to queue to cleanup worker when
	// process terminated.
	v.workers = append(v.workers, worker)
	ol.T(ctx, fmt.Sprintf("exec worker ok, pid=%v, total=%v", worker.pid, len(v.workers)))

	if err = v.checkWorkerApi(ctx, worker); err != nil {
		ol.E(ctx, "check worker api failed, err is", err)
		return
	}
	ol.T(ctx, "check srs worker api ok")

	if err = v.updateProxyApi(ctx, worker); err != nil {
		ol.E(ctx, "update proxy api failed, err is", err)
		return
	}
	ol.T(ctx, "exec process ok.")

	return
}

func (v *ShellBoss) checkWorkerApi(ctx ol.Context, worker *SrsWorker) (err error) {
	api := fmt.Sprintf("http://127.0.0.1:%v/api/v1/versions", worker.api)
	if err = checkApi(worker.cmd, api, processRetryMax, processExecInterval); err != nil {
		ol.E(ctx, fmt.Sprintf("srs failed, api=%v, max=%v, interval=%v, err is %v",
			api, processRetryMax, processExecInterval, err))
		return
	}

	// update current version
	if _, body, err := oh.ApiRequest(api); err != nil {
		ol.E(ctx, "request srs version failed, err is", err)
		return err
	} else {
		s := struct {
			Code int        `json:"code"`
			Data SrsVersion `json:"data"`
		}{}
		if err = json.Unmarshal(body, &s); err != nil {
			ol.E(ctx, "request srs version failed, err is", err)
			return err
		}

		worker.version = &s.Data
		ol.T(ctx, fmt.Sprintf("update version to %v, signature=%v",
			worker.version, worker.version.Signature))
	}

	return
}

func (v *ShellBoss) updateProxyApi(ctx ol.Context, worker *SrsWorker) (err error) {
	// notify rtmp and http proxy to update the active backend.
	url := fmt.Sprintf("http://127.0.0.1:%v/api/v1/proxy?rtmp=%v", v.conf.Rtmplb.Api, worker.rtmp)
	if _, _, err := oh.ApiRequest(url); err != nil {
		ol.E(ctx, "notify rtmp proxy failed, err is", err)
		return err
	}
	ol.T(ctx, "notify rtmp proxy ok, url is", url)

	url = fmt.Sprintf("http://127.0.0.1:%v/api/v1/proxy?http=%v", v.conf.Httplb.Api, worker.http)
	if _, _, err := oh.ApiRequest(url); err != nil {
		ol.E(ctx, "notify http proxy failed, err is", err)
		return err
	}
	ol.T(ctx, "notify http proxy ok, url is", url)

	backend := worker.api
	if v.conf.ApiProxyToBig() {
		backend = worker.big
	}
	url = fmt.Sprintf("http://127.0.0.1:%v/api/v1/proxy?port=%v", v.conf.Apilb.Api, backend)
	if _, _, err := oh.ApiRequest(url); err != nil {
		ol.E(ctx, "notify api proxy failed, err is", err)
		return err
	}
	ol.T(ctx, "notify api proxy ok, url is", url)

	return
}

func main() {
	var err error

	confFile := oo.ParseArgv("../conf/shell.json", kernel.Version(), signature)
	fmt.Println("SHELL is the process forker, config is", confFile)

	conf := &ShellConfig{}
	if err = conf.Loads(confFile); err != nil {
		fmt.Println("Loads config failed, err is", err)
		return
	}
	defer conf.Close()

	ctx := &kernel.Context{}
	ol.T(ctx, fmt.Sprintf("Config ok, %v", conf))

	shell := NewShellBoss(conf)

	if err = shell.ExecBuddies(ctx); err != nil {
		ol.E(ctx, "Shell exec buddies failed, err is", err)
		return
	}
	defer shell.Close()

	var apiListener net.Listener
	addrs := strings.Split(conf.Api, "://")
	apiNetwork, apiAddr := addrs[0], addrs[1]
	if apiListener, err = net.Listen(apiNetwork, apiAddr); err != nil {
		ol.E(ctx, "http listen failed, err is", err)
		return
	}
	defer apiListener.Close()

	oh.Server = signature
	wg := kernel.NewWorkerGroup()
	defer ol.T(ctx, "serve ok.")
	defer wg.Close()

	wg.QuitForSignals(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	// control messages
	wg.ForkGoroutine(func() {
		ctx := &kernel.Context{}
		ol.T(ctx, "Api: ready")
		defer ol.E(ctx, "Api: http handler ok")

		handler := http.NewServeMux()

		ol.T(ctx, fmt.Sprintf("Api: handle http://%v/api/v1/version", apiAddr))
		handler.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
			oh.WriteVersion(w, r, kernel.Version())
		})

		ol.T(ctx, fmt.Sprintf("Api: handle http://%v/api/v1/summary", apiAddr))
		handler.HandleFunc("/api/v1/summary", func(w http.ResponseWriter, r *http.Request) {
			oh.WriteVersion(w, r, kernel.Version())
		})

		ol.T(ctx, fmt.Sprintf("Api: handle http://%v/api/v1/upgrade", apiAddr))
		handler.HandleFunc("/api/v1/upgrade", func(w http.ResponseWriter, r *http.Request) {
			ctx := &kernel.Context{}
			if err = shell.Upgrade(ctx); err != nil {
				msg := fmt.Sprintf("upgrade failed, err is %v", err)
				oh.WriteCplxError(ctx, w, r, apiUpgradeError, msg)
				return
			}

			ol.T(ctx, "Api: upgrade ok.")
			oh.WriteData(ctx, w, r, nil)
		})

		server := &http.Server{Addr: apiAddr, Handler: handler}
		if err = server.Serve(apiListener); err != nil {
			if !wg.Closed() {
				ol.E(ctx, "Api: http serve failed, err is", err)
			}
			return
		}
	}, func() {
		apiListener.Close()
	})

	// cycle shell util quit.
	wg.ForkGoroutine(func() {
		ctx := &kernel.Context{}
		ol.T(ctx, "Cycle: ready")
		defer ol.T(ctx, "Cycle: terminated")

		shell.Cycle(ctx)
	}, func() {
		shell.Close()
	})

	// process singals
	go func() {
		ctx := &kernel.Context{}
		ol.T(ctx, "Signal: ready")
		defer ol.T(ctx, "Signal: signal ok.")

		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGUSR2)
		for _ = range c {
			// when upgrade failed, we serve as current workers.
			if err = shell.Upgrade(ctx); err != nil {
				ol.W(ctx, "Signal: upgrade failed, err is", err)
			} else {
				ol.T(ctx, "Signal: upgrade ok.")
			}
		}

	}()

	// wait for quit.
	wg.Wait()
	return
}
