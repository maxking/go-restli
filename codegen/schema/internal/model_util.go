package internal

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PapaCharlie/go-restli/codegen"
	"github.com/pkg/errors"
)

var (
	namespaceEscape  = regexp.MustCompile("([/.])_?internal([/.]?)")
	currentFile      string
	currentNamespace string
)

var (
	ModelRegistry = make(map[Identifier]*PdscModel)
	CyclicModels  = make(map[Identifier]bool)
)

func registerComplexType(t ComplexType) {
	id := t.GetIdentifier()
	if id.Namespace != "" && ModelRegistry[id] == nil {
		ModelRegistry[id] = &PdscModel{
			Type: t,
			File: currentFile,
		}
	}
}

func LoadModels() error {
	failedFiles := make(map[string]error)
	err := filepath.Walk(codegen.PdscDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		m := new(Model)
		err = codegen.ReadJSONFromFile(path, m)
		if err != nil {
			failedFiles[path] = err
			return nil
		}
		return nil
	})
	if err != nil {
		return err
	}

	failedFilesLength := len(failedFiles)
	for len(failedFiles) != 0 {
		for f := range failedFiles {
			m := new(Model)
			err = codegen.ReadJSONFromFile(f, m)
			if err != nil {
				failedFiles[f] = err
			} else {
				delete(failedFiles, f)
			}
		}

		if len(failedFiles) == failedFilesLength {
			return errors.Errorf("Failed to deserialize the following files: %+v", failedFiles)
		}
	}
	return nil
}

func trimUnneededModels(models []*Model) {
	loadedModels := make(map[Identifier]bool)
	for _, m := range models {
		if m.ComplexType != nil {
			loadedModels[m.ComplexType.GetIdentifier()] = true
		}
		for dep := range m.allDependencies(nil) {
			loadedModels[dep.GetIdentifier()] = true
		}
	}
	for id := range ModelRegistry {
		if !loadedModels[id] {
			delete(ModelRegistry, id)
		}
	}
}

func ResolveCyclicDependencies(loadedModels []*Model) {
	trimUnneededModels(loadedModels)

	for _, pdscModel := range ModelRegistry {
		m := pdscModel.toModel()
		for {
			modelChain := m.traverseDependencyGraph(nil, nil)
			if len(modelChain) > 0 {
				if modelChain[0].Name == modelChain[len(modelChain)-1].Name {
					log.Fatalf("%s depends on itself!", modelChain[0])
				} else {
					var identifiers []string
					for _, id := range modelChain {
						identifiers = append(identifiers, id.GetQualifiedClasspath())
					}

					log.Println("Detected cyclic dependency:", strings.Join(identifiers, " -> "))
				}
			} else {
				break
			}
		}

		dependsOnCyclicModel := false
		allDependencies := m.allDependencies(nil)
		for dep := range allDependencies {
			if CyclicModels[dep.GetIdentifier()] {
				dependsOnCyclicModel = true
				break
			}
		}
		if dependsOnCyclicModel {
			for dep := range allDependencies {
				CyclicModels[dep.GetIdentifier()] = true
			}
		}
	}
}

func (m *Model) traverseDependencyGraph(path []Identifier, visitedModels map[Identifier]bool) []Identifier {
	if path == nil && m.ComplexType != nil {
		path = []Identifier{m.ComplexType.GetIdentifier()}
	}
	if visitedModels == nil {
		visitedModels = map[Identifier]bool{}
	}

	for _, im := range m.innerModels() {
		innerPath := append([]Identifier(nil), path...)
		if im.ComplexType != nil && len(path) > 0 {
			startingModelId := path[0]
			previousModelId := path[len(path)-1]
			innerModelId := im.ComplexType.GetIdentifier()

			innerPath = append(innerPath, innerModelId)

			if visitedModels[innerModelId] || CyclicModels[innerModelId] {
				continue
			}

			if innerModelId.Namespace == startingModelId.Namespace && previousModelId.Namespace != innerModelId.Namespace {
				for _, id := range innerPath {
					CyclicModels[id] = true
				}
				return innerPath
			} else {
				visitedModels[innerModelId] = true
			}
		}

		if modelChain := im.traverseDependencyGraph(innerPath, visitedModels); len(modelChain) > 0 {
			return modelChain
		}
	}

	return nil
}

func (m *Model) allDependencies(types map[ComplexType]bool) map[ComplexType]bool {
	if types == nil {
		types = make(map[ComplexType]bool)
	}
	if m.ComplexType != nil {
		types[m.ComplexType] = true
	}
	for _, im := range m.innerModels() {
		if im.ComplexType != nil {
			if types[im.ComplexType] {
				break
			} else {
				types[im.ComplexType] = true
			}
		}
		for k, v := range im.allDependencies(types) {
			types[k] = v
		}
	}
	return types
}