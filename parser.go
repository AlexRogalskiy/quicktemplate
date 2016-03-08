package quicktemplate

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
)

func Parse(w io.Writer, r io.Reader) {
	s := NewScanner(r)
	parseTemplate(s, w)
}

func parseTemplate(s *Scanner, w io.Writer) {
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case Text:
			// just skip top-level text
		case TagName:
			switch string(t.Value) {
			case "code":
				parseCode(s, w)
			case "func":
				parseFunc(s, w)
			default:
				log.Fatalf("unexpected tag found outside func: %s at %s", t.Value, s.Pos())
			}
		default:
			log.Fatalf("unexpected token found when parsing template at %s: %s", s.Pos(), t)
		}
	}
	if err := s.LastError(); err != nil {
		log.Fatalf("cannot parse template: %s", err)
	}
}

func parseFunc(s *Scanner, w io.Writer) {
	t := expectTagContents(s)
	fOrig := string(t.Value)
	fname, fargs, fargsNoTypes := splitFnameFargs(s, fOrig)

	emitFuncStart(w, fname, fargs)
	for s.Next() {
		t := s.Token()
		switch t.ID {
		case Text:
			emitText(w, t.Value)
		case TagName:
			switch string(t.Value) {
			case "endfunc":
				expectTagContents(s)
				emitFuncEnd(w, fname, fargs, fargsNoTypes)
				return
			case "s":
				parseS(s, w)
			case "d":
				parseD(s, w)
			case "f":
				parseF(s, w)
			case "code":
				parseCode(s, w)
			default:
				log.Fatalf("unexpected tag found inside func: %s at %s", t.Value, s.Pos())
			}
		default:
			log.Fatalf("unexpected token found when parsing func at %s: %s", s.Pos(), t)
		}
	}
	if err := s.LastError(); err != nil {
		log.Fatalf("cannot parse func: %s", err)
	}
}

func emitFuncStart(w io.Writer, fname, fargs string) {
	fmt.Fprintf(w, "\nfunc %sStream(w *quicktemplate.Writer, %s) {\n", fname, fargs)
}

func emitFuncEnd(w io.Writer, fname, fargs, fargsNoTypes string) {
	fmt.Fprintf(w, "}\n\n")
	fmt.Fprintf(w, `func %s(%s) string {
	w := quicktemplate.AcquireWriter()
	%sStream(w, %s)
	s := w.String()
	quicktemplate.ReleaseWriter(w)
	return s
}`, fname, fargs, fname, fargsNoTypes)

}

func splitFnameFargs(s *Scanner, f string) (string, string, string) {
	// TODO: use real Go parser here

	n := strings.Index(f, "(")
	if n < 0 {
		log.Fatalf("missing '(' for function arguments at %s: %s", s.Pos(), f)
	}
	for n > 0 && isSpace(f[n-1]) {
		n--
	}
	fname := f[:n]

	fargs := f[n+1:]
	n = strings.LastIndex(fargs, ")")
	if n < 0 {
		log.Fatalf("missing ')' for function arguments at %s: %s", s.Pos(), f)
	}
	fargs = fargs[:n]

	var args []string
	for _, a := range strings.Split(fargs, ",") {
		n = 0
		for n < len(a) && isSpace(a[n]) {
			n++
		}
		a = a[n:]

		n = 0
		for n < len(a) && !isSpace(a[n]) {
			n++
		}

		args = append(args, a[:n])
	}
	fargsNoTypes := strings.Join(args, ", ")
	return fname, fargs, fargsNoTypes
}

func parseCode(s *Scanner, w io.Writer) {
	t := expectTagContents(s)
	fmt.Fprintf(w, "%s\n", t.Value)
}

func parseS(s *Scanner, w io.Writer) {
	t := expectTagContents(s)
	fmt.Fprintf(w, "w.E.S(%s)\n", t.Value)
}

func parseD(s *Scanner, w io.Writer) {
	t := expectTagContents(s)
	fmt.Fprintf(w, "w.D(%s)\n", t.Value)
}

func parseF(s *Scanner, w io.Writer) {
	t := expectTagContents(s)
	fmt.Fprintf(w, "w.F(%s)\n", t.Value)
}

func emitText(w io.Writer, text []byte) {
	for len(text) > 0 {
		n := bytes.IndexByte(text, '`')
		if n < 0 {
			fmt.Fprintf(w, "w.E.S(`%s`)\n", text)
			return
		}
		fmt.Fprintf(w, "w.E.S(`%s`)\n", text[:n])
		fmt.Fprintf(w, "w.E.S(\"`\")\n")
		text = text[n+1:]
	}
}

func expectTagContents(s *Scanner) *Token {
	return expectToken(s, TagContents)
}

func expectToken(s *Scanner, id int) *Token {
	if !s.Next() {
		log.Fatalf("cannot find token %s: %v", tokenIDToStr(id), s.LastError())
	}
	t := s.Token()
	if t.ID != id {
		log.Fatalf("unexpected token found %s. Expecting %s", t, tokenIDToStr(id))
	}
	return t
}