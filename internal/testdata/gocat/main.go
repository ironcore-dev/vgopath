// Copyright 2023 IronCore authors
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
	"io"
	"log"
	"os"
)

func main() {
	flag.Parse()

	filename := flag.Arg(0)
	if filename == "" {
		log.Fatalln("Must specify filename")
	}

	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Error opening file %q: %v\n", filename, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(os.Stdout, f); err != nil {
		log.Fatalf("Error copying file content: %v\n", err)
	}
}
