// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package link_test

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"

	. "github.com/ironcore-dev/vgopath/internal/link"
	"github.com/ironcore-dev/vgopath/internal/module"
)

var _ = Describe("Internal", func() {
	var (
		tmpDir                                                            string
		moduleA, moduleB, moduleB1, moduleB11, moduleB2, moduleC, moduleD module.Module
		allModules                                                        []module.Module
	)
	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "test")
		Expect(err).NotTo(HaveOccurred())

		allModules = []module.Module{}
		moduleA = module.Module{
			Path: "a",
			Dir:  filepath.Join("a"),
			Main: true,
		}
		allModules = append(allModules, moduleA)
		moduleB = module.Module{
			Path: "example.org/b",
			Dir:  filepath.Join("example.org", "b"),
		}
		allModules = append(allModules, moduleB)
		moduleB1 = module.Module{
			Path: "example.org/b/1",
			Dir:  filepath.Join("example.org", "b", "1"),
		}
		allModules = append(allModules, moduleB1)
		moduleB11 = module.Module{
			Path: "example.org/b/1/1",
			Dir:  filepath.Join("example.org", "b", "1", "1"),
		}
		allModules = append(allModules, moduleB11)
		moduleB2 = module.Module{
			Path: "example.org/b/2",
			Dir:  filepath.Join("example.org", "b", "2"),
		}
		allModules = append(allModules, moduleB2)
		moduleC = module.Module{
			Path: "example.org/user/c",
			Dir:  filepath.Join("example.org", "user", "c"),
		}
		allModules = append(allModules, moduleC)
		moduleD = module.Module{
			Path: "example.org/d",
		}
		allModules = append(allModules, moduleD)
	})
	AfterEach(func() {
		if tmpDir != "" {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		}
	})

	Describe("BuildModuleNodes", func() {
		It("should correctly build the nodes", func() {
			nodes, err := BuildModuleNodes([]module.Module{moduleA, moduleB, moduleB1, moduleB11, moduleB2, moduleC})
			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).To(ConsistOf(
				Node{
					Segment: "a",
					Module:  &moduleA,
				},
				Node{
					Segment: "example.org",
					Children: []Node{
						{
							Segment: "b",
							Module:  &moduleB,
							Children: []Node{
								{
									Segment: "1",
									Module:  &moduleB1,
									Children: []Node{
										{
											Segment: "1",
											Module:  &moduleB11,
										},
									},
								},
								{
									Segment: "2",
									Module:  &moduleB2,
								},
							},
						},
						{
							Segment: "user",
							Children: []Node{
								{
									Segment: "c",
									Module:  &moduleC,
								},
							},
						},
					},
				},
			))
		})

		It("should error on invalid module paths", func() {
			_, err := BuildModuleNodes([]module.Module{{Path: ""}})
			Expect(err).To(HaveOccurred())
		})

		It("should error if there are modules pointing to the same path", func() {
			_, err := BuildModuleNodes([]module.Module{{Path: "foo"}, {Path: "foo"}})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("FilterModulesWithoutDir", func() {
		It("should correctly filter the modules", func() {
			mods := FilterModulesWithoutDir([]module.Module{moduleA, moduleB, moduleD})
			Expect(mods).To(Equal([]module.Module{moduleA, moduleB}))
		})
	})

	Context("Link", func() {
		var (
			srcGopathDir string
			dstGopathDir string
		)
		BeforeEach(func() {
			srcGopathDir = filepath.Join(tmpDir, "srcGopath")
			dstGopathDir = filepath.Join(tmpDir, "dstGopath")

			Expect(os.MkdirAll(srcGopathDir, 0777)).To(Succeed())
			Expect(os.MkdirAll(dstGopathDir, 0777)).To(Succeed())
		})

		Describe("Nodes", func() {
			It("should correctly handle submodules", func() {
				Expect(makeModules(srcGopathDir, &moduleB, &moduleB1, &moduleB11, &moduleB2)).NotTo(HaveOccurred())

				nodes, err := BuildModuleNodes([]module.Module{moduleB, moduleB1, moduleB11, moduleB2})
				Expect(err).NotTo(HaveOccurred())

				Expect(Nodes(dstGopathDir, nodes)).To(Succeed())

				Expect(dstGopathDir).To(HaveEntries(map[string]types.GomegaMatcher{
					filepath.Join("example.org", "b"):                     BeADirectory(),
					filepath.Join("example.org", "b", "go.mod"):           BeASymlinkTo(filepath.Join(moduleB.Dir, "go.mod")),
					filepath.Join("example.org", "b", "1"):                BeADirectory(),
					filepath.Join("example.org", "b", "1", "go.mod"):      BeASymlinkTo(filepath.Join(moduleB1.Dir, "go.mod")),
					filepath.Join("example.org", "b", "1", "1"):           BeADirectory(),
					filepath.Join("example.org", "b", "1", "1", "go.mod"): BeASymlinkTo(filepath.Join(moduleB11.Dir, "go.mod")),
					filepath.Join("example.org", "b", "2"):                BeADirectory(),
					filepath.Join("example.org", "b", "2", "go.mod"):      BeASymlinkTo(filepath.Join(moduleB2.Dir, "go.mod")),
				}))
			})
		})

		Describe("GoBin", func() {
			var (
				srcGoBinDir string
				dstGoBinDir string
			)
			BeforeEach(func() {
				srcGoBinDir = filepath.Join(srcGopathDir, "bin")
				Expect(os.MkdirAll(srcGoBinDir, 0777)).To(Succeed())

				dstGoBinDir = filepath.Join(dstGopathDir, "bin")
				Expect(os.MkdirAll(dstGopathDir, 0777)).To(Succeed())
			})

			It("should correctly link go bin", func() {
				defer setEnvAndRevert("GOBIN", "")()
				defer setAndRevert(&build.Default.GOPATH, srcGopathDir)()

				Expect(GoBin(dstGopathDir)).To(Succeed())
				Expect(dstGoBinDir).To(BeASymlinkTo(srcGoBinDir))
			})

			It("should correctly link go bin if GOBIN is set", func() {
				defer setEnvAndRevert("GOBIN", srcGoBinDir)()

				Expect(GoBin(dstGopathDir)).To(Succeed())
				Expect(dstGoBinDir).To(BeASymlinkTo(srcGoBinDir))
			})
		})

		Describe("GoPkg", func() {
			var (
				srcGoPkgDir string
				dstGoPkgDir string
			)
			BeforeEach(func() {
				srcGoPkgDir = filepath.Join(srcGopathDir, "pkg")
				Expect(os.MkdirAll(srcGoPkgDir, 0777)).To(Succeed())

				dstGoPkgDir = filepath.Join(dstGopathDir, "pkg")
				Expect(os.MkdirAll(dstGopathDir, 0777)).To(Succeed())
			})

			It("should correctly link go pkg", func() {
				defer setAndRevert(&build.Default.GOPATH, srcGopathDir)()

				Expect(GoPkg(dstGopathDir)).To(Succeed())
				Expect(dstGoPkgDir).To(BeASymlinkTo(srcGoPkgDir))
			})
		})
	})
})

func makeModules(gopath string, mods ...*module.Module) error {
	for _, mod := range mods {
		// update dir to include gopath prefix
		mod.Dir = filepath.Join(gopath, mod.Dir)

		if err := os.MkdirAll(mod.Dir, 0777); err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(mod.Dir, "go.mod"), []byte("module "+mod.Path+"\n"), 0666); err != nil {
			return err
		}
	}
	return nil
}

func HaveEntries(expected map[string]types.GomegaMatcher) types.GomegaMatcher {
	return &haveEntriesMatcher{matchers: expected}
}

// haveEntriesMatcher is very similar to matchers.AndMatcher.
type haveEntriesMatcher struct {
	matchers map[string]types.GomegaMatcher

	// state
	baseDir             string
	firstFailedFilename string
}

func (m *haveEntriesMatcher) Match(actual interface{}) (success bool, err error) {
	m.firstFailedFilename = ""

	var ok bool
	m.baseDir, ok = actual.(string)
	if !ok {
		return false, fmt.Errorf("HaveEntries matcher expects a string but got %T", actual)
	}

	// sort matchers by filename for stable test results even though maps are unsorted
	filenames := make([]string, 0, len(m.matchers))
	for filename := range m.matchers {
		filenames = append(filenames, filename)
	}
	slices.Sort(filenames)

	for _, filename := range filenames {
		matcher := m.matchers[filename]

		success, err := matcher.Match(filepath.Join(m.baseDir, filename))
		if !success || err != nil {
			m.firstFailedFilename = filename
			return false, err
		}
	}

	return true, nil
}

func (m *haveEntriesMatcher) FailureMessage(actual interface{}) (message string) {
	return m.matchers[m.firstFailedFilename].FailureMessage(filepath.Join(m.baseDir, m.firstFailedFilename))
}

func (m *haveEntriesMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	// not the most beautiful list of matchers, but not bad either...
	return format.Message(actual, "not to have these entries: %s", m.matchers)
}

func BeASymlinkTo(filename string) types.GomegaMatcher {
	return &beASymlinkToMatcher{filename}
}

type beASymlinkToMatcher struct {
	filename string
}

func (m *beASymlinkToMatcher) Match(actual interface{}) (success bool, err error) {
	actualFilename, ok := actual.(string)
	if !ok {
		return false, fmt.Errorf("IsSymlinkTo expects a filename string")
	}

	actualStat, err := os.Lstat(actualFilename)
	if err != nil {
		return false, err
	}

	if (actualStat.Mode() & os.ModeSymlink) != os.ModeSymlink {
		return false, nil
	}

	tgt, err := os.Readlink(actualFilename)
	if err != nil {
		return false, err
	}

	return tgt == m.filename, nil
}

func (m *beASymlinkToMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%v\nto be a symlink to\n\t%s", actual, m.filename)
}

func (m *beASymlinkToMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%v\nnot to be a symlink to\n\t%s", actual, m.filename)
}

func setEnvAndRevert(key, value string) func() {
	oldValue := os.Getenv(key)
	if value == "" {
		_ = os.Unsetenv(key)
	} else {
		_ = os.Setenv(key, value)
	}
	return func() {
		if oldValue == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, oldValue)
		}
	}
}

func setAndRevert(pointerToString *string, newValue string) func() {
	oldValue := *pointerToString
	*pointerToString = newValue
	return func() {
		*pointerToString = oldValue
	}
}
