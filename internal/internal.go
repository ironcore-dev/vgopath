// Copyright 2022 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"encoding/json"
	"fmt"
	"go/build"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Node struct {
	Segment  string
	Module   *Module
	Children []Node
}

func insertModuleInNode(node *Node, mod Module, relativeSegments []string) error {
	if len(relativeSegments) == 0 {
		if node.Module != nil {
			return fmt.Errorf("cannot insert module %s into node %s: module %s already exists", mod.Path, node.Segment, node.Module.Path)
		}

		node.Module = &mod
		return nil
	}

	var (
		idx     = -1
		segment = relativeSegments[0]
	)
	for i, child := range node.Children {
		if child.Segment == segment {
			idx = i
			break
		}
	}

	var child *Node
	if idx == -1 {
		child = &Node{Segment: segment}
	} else {
		child = &node.Children[idx]
	}

	if err := insertModuleInNode(child, mod, relativeSegments[1:]); err != nil {
		return err
	}

	if idx == -1 {
		node.Children = append(node.Children, *child)
	}

	return nil
}

func BuildModuleNodes(modules []Module) ([]Node, error) {
	sort.Slice(modules, func(i, j int) bool { return modules[i].Path < modules[j].Path })
	nodeByRootSegment := make(map[string]*Node)

	for _, module := range modules {
		if module.Path == "" {
			return nil, fmt.Errorf("invalid empty module path")
		}

		segments := strings.Split(module.Path, "/")

		rootSegment := segments[0]
		node, ok := nodeByRootSegment[rootSegment]
		if !ok {
			node = &Node{Segment: rootSegment}
			nodeByRootSegment[rootSegment] = node
		}

		if err := insertModuleInNode(node, module, segments[1:]); err != nil {
			return nil, err
		}
	}

	res := make([]Node, 0, len(nodeByRootSegment))
	for _, node := range nodeByRootSegment {
		res = append(res, *node)
	}
	return res, nil
}

type Module struct {
	Path    string
	Dir     string
	Version string
	Main    bool
}

type moduleReader struct {
	mu sync.Mutex

	cmd    *exec.Cmd
	stdout io.ReadCloser

	exited  bool
	waitErr error
}

func StartGoModListJSONReader(dir string) (io.ReadCloser, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	cmd.Dir = dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &moduleReader{
		cmd:    cmd,
		stdout: stdout,
	}, nil
}

func (r *moduleReader) Read(p []byte) (n int, err error) {
	return r.stdout.Read(p)
}

func (r *moduleReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.exited {
		return r.waitErr
	}

	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		r.waitErr = r.cmd.Wait()
	}()

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		return fmt.Errorf("error waiting for command to be completed")
	case <-waitDone:
		return r.waitErr
	}
}

func ParseModules(r io.Reader) ([]Module, error) {
	var (
		mods []Module
		dec  = json.NewDecoder(r)
	)

	for {
		var mod Module
		if err := dec.Decode(&mod); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		mods = append(mods, mod)
	}
	return mods, nil
}

func ReadModules() ([]Module, error) {
	rc, err := StartGoModListJSONReader("")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	mods, err := ParseModules(rc)
	if err != nil {
		return nil, fmt.Errorf("error parsing modules: %w", err)
	}

	return mods, nil
}

func FilterVendorModules(modules []Module) []Module {
	var res []Module
	for _, module := range modules {
		// Don't vendor modules without paths / main modules.
		if module.Dir == "" || module.Version == "" && module.Main {
			continue
		}

		res = append(res, module)
	}

	return res
}

type Options struct {
	SkipGoBin bool
	SkipGoSrc bool
	SkipGoPkg bool
}

func Run(dstDir string, opts Options) error {
	if !opts.SkipGoSrc {
		if err := LinkGoSrc(dstDir); err != nil {
			return fmt.Errorf("error linking GOPATH/src: %w", err)
		}
	}

	if !opts.SkipGoBin {
		if err := LinkGoBin(dstDir); err != nil {
			return fmt.Errorf("error linking GOPATH/bin: %w", err)
		}
	}

	if !opts.SkipGoPkg {
		if err := LinkGoPkg(dstDir); err != nil {
			return fmt.Errorf("error linking GOPATH/pkg: %w", err)
		}
	}

	return nil
}

func LinkGoBin(dstDir string) error {
	dstGoBinDir := filepath.Join(dstDir, "bin")
	if err := os.RemoveAll(dstGoBinDir); err != nil {
		return err
	}

	srcGoBinDir := os.Getenv("GOBIN")
	if srcGoBinDir == "" {
		srcGoBinDir = filepath.Join(build.Default.GOPATH, "bin")
	}

	if err := os.Symlink(srcGoBinDir, dstGoBinDir); err != nil {
		return err
	}
	return nil
}

func LinkGoPkg(dstDir string) error {
	dstGoPkgDir := filepath.Join(dstDir, "pkg")
	if err := os.RemoveAll(dstGoPkgDir); err != nil {
		return err
	}

	if err := os.Symlink(filepath.Join(build.Default.GOPATH, "pkg"), dstGoPkgDir); err != nil {
		return err
	}
	return nil
}

func LinkGoSrc(dstDir string) error {
	mods, err := ReadModules()
	if err != nil {
		return fmt.Errorf("error reading modules: %w", err)
	}

	mods = FilterVendorModules(mods)

	nodes, err := BuildModuleNodes(mods)
	if err != nil {
		return fmt.Errorf("error building module tree: %w", err)
	}

	dstGoSrcDir := filepath.Join(dstDir, "src")
	if err := os.RemoveAll(dstGoSrcDir); err != nil {
		return err
	}

	if err := os.Mkdir(dstGoSrcDir, 0777); err != nil {
		return err
	}

	if err := LinkNodes(dstGoSrcDir, nodes); err != nil {
		return err
	}
	return nil
}

type linkNodeError struct {
	path string
	err  error
}

func (l *linkNodeError) Error() string {
	return fmt.Sprintf("[path %s]: %v", l.path, l.err)
}

func joinLinkNodeError(node Node, err error) error {
	if linkNodeErr, ok := err.(*linkNodeError); ok {
		return &linkNodeError{
			path: path.Join(node.Segment, linkNodeErr.path),
			err:  linkNodeErr.err,
		}
	}
	return &linkNodeError{
		path: node.Segment,
		err:  err,
	}
}

func LinkNodes(dir string, nodes []Node) error {
	for _, node := range nodes {
		if err := linkNode(dir, node); err != nil {
			return joinLinkNodeError(node, err)
		}
	}
	return nil
}

func linkNode(dir string, node Node) error {
	dstDir := filepath.Join(dir, node.Segment)

	// If the node specifies a module and no children are present, we can take optimize and directly
	// symlink the module directory to the destination directory.
	if node.Module != nil && len(node.Children) == 0 {
		srcDir := node.Module.Dir

		if err := os.Symlink(srcDir, dstDir); err != nil {
			return fmt.Errorf("error symlinking node: %w", err)
		}
	}

	if err := os.RemoveAll(dstDir); err != nil {
		return err
	}

	if err := os.Mkdir(dstDir, 0777); err != nil {
		return err
	}

	if node.Module != nil {
		srcDir := node.Module.Dir
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			srcPath := filepath.Join(srcDir, entry.Name())
			dstPath := filepath.Join(dstDir, entry.Name())
			if err := os.Symlink(srcPath, dstPath); err != nil {
				return fmt.Errorf("error symlinking entry %s to %s: %w", srcPath, dstPath, err)
			}
		}
	}
	return LinkNodes(dstDir, node.Children)
}
