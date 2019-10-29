package models

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

type typeField struct {
	Type string `json:"type"`
}

type docField struct {
	Doc string `json:"doc"`
}

type WrongTypeError struct {
	Expected, Actual string
}

func (w *WrongTypeError) Error() string {
	return fmt.Sprintf("models: Incorrect type, expected %s got %s", w.Expected, w.Actual)
}

type Model struct {
	BuiltinType BuiltinType
	ComplexType ComplexType

	namespace string
	ref       *ModelReference
}

func (m *Model) String() string {
	var t string
	var s interface{}
	if m.BuiltinType != nil {
		t = "BuiltinType"
		s = m.BuiltinType
	}

	if m.ComplexType != nil {
		t = "ComplexType"
		s = m.ComplexType
	}

	if m.ref != nil {
		t = "Ref"
		s = m.ref
	}

	return fmt.Sprintf("Model{namespace: %s, %s: %+v}", m.namespace, t, s)
}

type hasInnerModels interface {
	innerModels() []*Model
}

func (m *Model) innerModels() []*Model {
	if im, ok := m.ComplexType.(hasInnerModels); ok {
		return im.innerModels()
	}
	if im, ok := m.BuiltinType.(hasInnerModels); ok {
		return im.innerModels()
	}
	return nil
}

func (m *Model) UnmarshalJSON(data []byte) error {
	defer func() {
		m.register()
		m.propagateNamespaces()
		m.replaceRef()
	}()

	model := &struct {
		Namespace string `json:"namespace"`
		Type      json.RawMessage
	}{}

	if err := json.Unmarshal(data, model); err != nil {
		var unmarshalErrors []error
		var subErr error

		var bytes BytesModel
		if subErr = json.Unmarshal(data, &bytes); subErr == nil {
			m.BuiltinType = &bytes
			return nil
		} else {
			unmarshalErrors = append(unmarshalErrors, subErr)
		}

		var primitive PrimitiveModel
		if subErr = json.Unmarshal(data, &primitive); subErr == nil {
			m.BuiltinType = &primitive
			return nil
		} else {
			unmarshalErrors = append(unmarshalErrors, subErr)
		}

		var reference ModelReference
		if subErr = json.Unmarshal(data, &reference); subErr == nil {
			m.ref = &reference
			return nil
		} else {
			unmarshalErrors = append(unmarshalErrors, subErr)
		}

		union := &UnionModel{}
		if subErr = json.Unmarshal(data, union); subErr == nil {
			m.BuiltinType = union
			return nil
		} else {
			unmarshalErrors = append(unmarshalErrors, subErr)
		}

		return errors.Errorf("illegal model type: %v, %v, (%s)", unmarshalErrors, err, string(data))
	}

	m.namespace = model.Namespace

	var modelType string
	if err := json.Unmarshal(model.Type, &modelType); err != nil {
		return errors.Wrap(err, "type must either be a string or union")
	}

	switch modelType {
	case RecordTypeModelTypeName:
		recordType := &RecordModel{}
		if err := json.Unmarshal(data, recordType); err == nil {
			m.ComplexType = recordType
			return nil
		} else {
			return errors.WithStack(err)
		}
	case EnumModelTypeName:
		enumType := &EnumModel{}
		if err := json.Unmarshal(data, enumType); err == nil {
			m.ComplexType = enumType
			return nil
		} else {
			return errors.WithStack(err)
		}
	case FixedModelTypeName:
		fixedType := &FixedModel{}
		if err := json.Unmarshal(data, fixedType); err == nil {
			m.ComplexType = fixedType
			return nil
		} else {
			return errors.WithStack(err)
		}
	case MapModelTypeName:
		mapType := &MapModel{}
		if err := json.Unmarshal(data, mapType); err == nil {
			m.BuiltinType = mapType
			return nil
		} else {
			return errors.WithStack(err)
		}
	case ArrayModelTypeName:
		arrayType := &ArrayModel{}
		if err := json.Unmarshal(data, arrayType); err == nil {
			m.BuiltinType = arrayType
			return nil
		} else {
			return errors.WithStack(err)
		}
	case TyperefModelTypeName:
		typerefType := &TyperefModel{}
		if err := json.Unmarshal(data, typerefType); err == nil {
			m.ComplexType = typerefType
			return nil
		} else {
			return errors.WithStack(err)
		}
	case BytesModelTypeName:
		m.BuiltinType = &BytesModel{}
		return nil
	}

	var primitiveType PrimitiveModel
	if err := json.Unmarshal(model.Type, &primitiveType); err == nil {
		m.BuiltinType = &primitiveType
		return nil
	}

	var referenceType ModelReference
	if err := json.Unmarshal(model.Type, &referenceType); err == nil {
		m.ref = &referenceType
		return nil
	}

	return errors.Errorf("could not deserialize %v into %v", string(data), m)
}

func (m *Model) register() {
	if m.ComplexType != nil {
		id := m.ComplexType.GetIdentifier()
		if id.Namespace != "" && ModelCache[id] == nil {
			ModelCache[id] = m.ComplexType
		}
	}
}

func (m *Model) propagateNamespaces() {
	if m.ref != nil {
		if m.namespace != "" && m.ref.Namespace == "" {
			m.ref.Namespace = m.namespace
		}
	}

	for _, child := range m.innerModels() {
		if child.namespace == "" {
			child.namespace = m.namespace
		}
		child.propagateNamespaces()
	}
}

func (m *Model) replaceRef() {
	if m.ref != nil {
		if resolvedModel := m.ref.Resolve(); resolvedModel != nil {
			m.ComplexType = resolvedModel
			m.ref = nil
		}
	}

	for _, child := range m.innerModels() {
		child.replaceRef()
	}
}