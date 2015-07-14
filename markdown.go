package ablog

import (
	"bytes"
	"github.com/russross/blackfriday"
	"os/exec"
)

const commonHtmlFlags = 0 |
	blackfriday.HTML_USE_XHTML |
	blackfriday.HTML_USE_SMARTYPANTS |
	blackfriday.HTML_SMARTYPANTS_FRACTIONS |
	blackfriday.HTML_SMARTYPANTS_LATEX_DASHES

const commonExtensions = 0 |
	blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
	blackfriday.EXTENSION_TABLES |
	blackfriday.EXTENSION_FENCED_CODE |
	blackfriday.EXTENSION_AUTOLINK |
	blackfriday.EXTENSION_STRIKETHROUGH |
	blackfriday.EXTENSION_SPACE_HEADERS |
	blackfriday.EXTENSION_HEADER_IDS

type Renderer struct {
	*blackfriday.Html
}

func (r *Renderer) BlockCode(out *bytes.Buffer, text []byte, lang string) {
	if len(lang) == 0 {
		r.Html.BlockCode(out, text, lang)
		return
	}

	var stderr bytes.Buffer

	cmd := exec.Command("pygmentize", "-l"+lang, "-fhtml")
	cmd.Stdin = bytes.NewReader(text)
	cmd.Stdout = out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		exitErr(err, "Failed to run pygmentize.")
	}
}

func markdown(input []byte) []byte {
	renderer := &Renderer{
		Html: blackfriday.HtmlRenderer(
			commonHtmlFlags, "", "").(*blackfriday.Html),
	}
	return blackfriday.Markdown(input, renderer, commonExtensions)
}
