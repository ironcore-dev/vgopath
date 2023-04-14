// Copyright 2023 OnMetal authors
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

package testdata

import "github.com/onmetal/vgopath/internal/module"

var (
	ModuleA = module.Module{
		Path: "a",
		Dir:  "/tmp/a",
		Main: true,
	}

	ModuleB = module.Module{
		Path: "example.org/b",
		Dir:  "/tmp/example.org/b",
	}

	ModuleD = module.Module{
		Path: "example.org/d",
	}

	Modules = []module.Module{
		ModuleA,
		ModuleB,
		ModuleD,
	}
)
