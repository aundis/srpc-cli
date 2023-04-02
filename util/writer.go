package util

import (
	"bytes"
)

type TextWriter interface {
	Line() int
	Column() int
	Indent() int
	TextPos() int

	RawWrite(s string)
	Write(p []byte) TextWriter
	WriteString(s ...string) TextWriter
	WriteLine() TextWriter
	WriteEmptyLine() TextWriter

	IncreaseIndent() TextWriter
	DecreaseIndent() TextWriter

	String() string
	Bytes() []byte
}

type textWriter struct {
	newLine   string
	output    bytes.Buffer
	indent    int
	lineStart bool
	lineCount int
	linePos   int
}

func NewTextWriter() TextWriter {
	return &textWriter{
		newLine:   "\n",
		indent:    0,
		lineStart: true,
		lineCount: 0,
		linePos:   0,
	}
}

var indentStrings = []string{"", "    "}

func getIndentString(level int) string {
	if level >= len(indentStrings) {
		indentStrings = append(indentStrings, getIndentString(level-1)+indentStrings[1])
	}
	return indentStrings[level]
}

func getIndentSize() int {
	return len(indentStrings[1])
}

func (w *textWriter) Write(p []byte) TextWriter {
	if len(p) > 0 {
		if w.lineStart {
			w.output.WriteString(getIndentString(w.indent))
			w.lineStart = false
		}
		w.output.Write(p)
	}
	return w
}

func (w *textWriter) WriteString(arr ...string) TextWriter {
	for _, s := range arr {
		if len(s) == 0 {
			continue
		}
		if w.lineStart {
			w.output.WriteString(getIndentString(w.indent))
			w.lineStart = false
		}
		w.output.WriteString(s)
	}
	return w
}

func (w *textWriter) WriteLine() TextWriter {
	if !w.lineStart {
		w.output.WriteString(w.newLine)
		w.lineCount++
		w.linePos = len(w.output.String())
		w.lineStart = true
	}
	return w
}

func (w *textWriter) WriteEmptyLine() TextWriter {
	w.output.WriteString(w.newLine)
	w.lineCount++
	w.linePos = len(w.output.String())
	w.lineStart = true
	return w
}

func (w *textWriter) IncreaseIndent() TextWriter {
	w.indent++
	return w
}

func (w *textWriter) DecreaseIndent() TextWriter {
	w.indent--
	return w
}

func (w *textWriter) String() string {
	return w.output.String()
}

func (w *textWriter) Bytes() []byte {
	return w.output.Bytes()
}

func (w *textWriter) RawWrite(s string) {
	if len(s) > 0 {
		if w.lineStart {
			w.lineStart = false
		}
		w.output.WriteString(s)
	}
}

func (w *textWriter) TextPos() int {
	return len(w.output.String())
}

func (w *textWriter) Line() int {
	return w.lineCount + 1
}

func (w *textWriter) Column() int {
	if w.lineStart {
		return w.indent*getIndentSize() + 1
	} else {
		return len(w.output.String()) - w.linePos + 1
	}
}

func (w *textWriter) Indent() int {
	return w.indent
}
