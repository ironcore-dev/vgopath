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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/onmetal/vgopath/internal"
)

func Usage() {
	_, _ = fmt.Fprint(flag.CommandLine.Output(), `vgopath - Virtual gopath

Usage:
	vgopath <dir> [opts]

Create a 'virtual' GOPATH at the specified directory.
Has to be run from a go module.

vgopath will setup a GOPATH folder structure, ensuring that any tool used
to the traditional setup will function as normal.

The current module will be mirrored to where its go.mod path (the line
after 'module') points at.

`)
	flag.PrintDefaults()
}

func main() {
	var opts internal.Options
	flag.BoolVar(&opts.SkipGoPkg, "skip-go-pkg", opts.SkipGoPkg, "Whether to skip mirroring $GOPATH/pkg")
	flag.BoolVar(&opts.SkipGoBin, "skip-go-bin", opts.SkipGoBin, "Whether to skip mirroring $GOBIN")
	flag.BoolVar(&opts.SkipGoSrc, "skip-go-src", opts.SkipGoSrc, "Whether to skip mirroring modules as src")
	flag.Usage = Usage

	flag.Parse()
	dstDir := flag.Arg(0)
	if dstDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	if err := internal.Run(dstDir, opts); err != nil {
		fmt.Printf("Error running vgopath:\n%v", err)
		os.Exit(1)
	}
}
