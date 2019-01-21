package codegen

import (
	"bytes"
	"github.com/dave/jennifer/jen"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var CommentWrapWidth = 120

type CodeFile struct {
	PackagePath string
	Filename    string
	Code        *jen.Statement
}

func (f *CodeFile) Write(outputDir string) (filename string, err error) {
	file := jen.NewFilePath(f.PackagePath)
	file.Add(f.Code)
	filename = filepath.Join(outputDir, f.PackagePath, f.Filename+".go")

	err = write(filename, file)
	return
}

func write(filename string, file *jen.File) error {
	b := bytes.NewBuffer(nil)
	if err := file.Render(b); err != nil {
		return errors.WithStack(err)
	}

	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return errors.WithStack(err)
	}

	if err := ioutil.WriteFile(filename, b.Bytes(), os.ModePerm); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func AddWordWrappedComment(code *jen.Statement, comment string) *jen.Statement {
	if comment != "" {
		code.Comment(comment)
		return code
	} else {
		return code
	}

	// WIP: Restli comments are not behaving quite as expected, so comments get added as is, without being wrapped
	for len(comment) > CommentWrapWidth {
		if newline := strings.Index(comment[:CommentWrapWidth], "\n"); newline != -1 {
			code.Comment(comment[:newline]).Line()
			comment = comment[newline+1:]
			continue
		}

		if index := strings.LastIndexFunc(comment[:CommentWrapWidth], unicode.IsSpace); index > 0 {
			code.Comment(comment[:index]).Line()
			comment = comment[index+1:]
		} else {
			break
		}
	}

	code.Comment(comment)

	return code
}

func ExportedIdentifier(identifier string) string {
	return strings.ToUpper(identifier[:1]) + identifier[1:]
}

func JsonTag(fieldName string) map[string]string {
	return map[string]string{"json": fieldName}
}