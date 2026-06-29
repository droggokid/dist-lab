package tui

import (
	"fmt"
	"os"

	"dist-lab/internal/input"
)

func (m *Model) resetLoadedData() {
	m.parser = nil
	m.filePaths = nil
	m.fileSizes = nil
	m.selectedPath = ""
	m.fieldCount = 0
	m.docCount = 0
	m.fields = fieldsModel{}
	m.clearValues()
}

func (m *Model) ensureParser() {
	if m.parser != nil {
		return
	}

	m.parser = input.NewParser()
	m.filePaths = []string{}
	m.fileSizes = []int64{}
	m.selectedPath = ""
	m.clearValues()
}

func (m *Model) loadFile(path string) error {
	m.ensureParser()

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat file %q: %w", path, err)
	}

	loaded := input.NewParser()
	if err := loaded.AddFile(path); err != nil {
		return err
	}
	if len(m.parser.Fields) == 0 && len(loaded.Fields) == 0 {
		return fmt.Errorf("no fields found in combined files")
	}

	m.parser.Merge(loaded)
	m.filePaths = append(m.filePaths, path)
	m.fileSizes = append(m.fileSizes, info.Size())
	m.fieldCount = len(m.parser.Fields)
	m.docCount = len(m.parser.Docs)

	m.fields = newFieldsModel(m.parser.Fields)
	return nil
}
