package web

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/gomarkdown/markdown/ast"

	"io"
)

func mdToHTML(md []byte) []byte {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs |
				parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

func streamMD(w io.Writer, doc ast.Node, renderer markdown.Renderer) error {
	renderer.RenderHeader(w, doc)
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		return renderer.RenderNode(w, node, entering)
	})
	renderer.RenderFooter(w, doc)
	return nil
}

func readmeMarkdown(md []byte, w io.Writer) error {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs |
				parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return streamMD(w, doc, renderer)
}

func textToHTML(r io.Reader, w io.Writer) error {
	w.Write([]byte("<div class=\"plain-text\">"))
	var data [1024]byte
	for {
		n, err := r.Read(data[:])
		if err != nil { return err }
		j := 0
		for i := 0; i < n; i++ {
			switch data[i] {
			case '<', '>', '&', '\'', '"': break
			default: continue
			}
			w.Write(data[j:i])
			j = i + 1
			switch data[i] {
			case '<': w.Write([]byte("&lt;"))
			case '>': w.Write([]byte("&gt;"))
			case '&': w.Write([]byte("&amp;"))
			case '\'': w.Write([]byte("&#39;"))
			case '"': w.Write([]byte("&quot;"))
			}
		}
		w.Write(data[j:n])
		if n != 1024 { break }
	}
	_, err := w.Write([]byte("</div>"))
	return err
}
