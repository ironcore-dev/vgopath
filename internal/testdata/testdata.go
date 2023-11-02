// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package testdata

import "github.com/ironcore-dev/vgopath/internal/module"

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
