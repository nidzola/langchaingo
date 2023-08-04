package documentloaders

import (
	"context"
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"

	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

// PDF loads text data from an io.Reader.
type PDF struct {
	r        io.Reader
	s        int64
	password string
}

var _ Loader = PDF{}

// PDFOptions are options for the PDF loader.
type PDFOptions func(pdf *PDF)

// WithPassword sets the password for the PDF.
func WithPassword(password string) PDFOptions {
	return func(pdf *PDF) {
		pdf.password = password
	}
}

// NewPDF creates a new text loader with an io.Reader.
func NewPDF(r io.Reader, size int64, opts ...PDFOptions) PDF {
	_pdf := PDF{
		r: r,
		s: size,
	}
	for _, opt := range opts {
		opt(&_pdf)
	}
	return _pdf
}

// getPassword returns the password for the PDF
// it than clears the password on the struct, so it can't be used again
// if the password is cleared and tried to be used again it will fail.
func (p *PDF) getPassword() string {
	pass := p.password
	p.password = ""
	return pass
}

// Load reads from the io.Reader for the PDF data and returns the documents with the data and with
// metadata attached of the page number and total number of pages of the PDF.
func (p PDF) Load(_ context.Context) ([]schema.Document, error) {
	var reader *pdf.Reader
	var err error

	// converting io.Reader to io.ReaderAt
	readerAt, err := newBufferedReaderAt(p.r)
	if err != nil {
		return nil, err
	}

	if p.password != "" {
		reader, err = pdf.NewReaderEncrypted(readerAt, p.s, p.getPassword)
		if err != nil {
			return nil, err
		}
	} else {
		reader, err = pdf.NewReader(readerAt, p.s)
		if err != nil {
			return nil, err
		}
	}

	numPages := reader.NumPage()

	docs := []schema.Document{}

	// fonts to be used when getting plain text from pages
	fonts := make(map[string]*pdf.Font)
	for i := 1; i < numPages+1; i++ {
		p := reader.Page(i)
		// add fonts to map
		for _, name := range p.Fonts() {
			// only add the font if we don't already have it
			if _, ok := fonts[name]; !ok {
				f := p.Font(name)
				fonts[name] = &f
			}
		}
		text, err := p.GetPlainText(fonts)
		if err != nil {
			return nil, err
		}

		// add the document to the doc list
		docs = append(docs, schema.Document{
			PageContent: text,
			Metadata: map[string]any{
				"page":        i,
				"total_pages": numPages,
			},
		})
	}

	return docs, nil
}

// LoadAndSplit reads pdf data from the io.Reader and splits it into multiple
// documents using a text splitter.
func (p PDF) LoadAndSplit(ctx context.Context, splitter textsplitter.TextSplitter) ([]schema.Document, error) {
	docs, err := p.Load(ctx)
	if err != nil {
		return nil, err
	}

	return textsplitter.SplitDocuments(splitter, docs)
}

type bufferedReaderAt struct {
	data   []byte
	offset int64
}

// newBufferedReaderAt wrapper to convert io.Reader to io.ReaderAa
func newBufferedReaderAt(reader io.Reader) (*bufferedReaderAt, error) {
	buf, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return &bufferedReaderAt{
		data:   buf,
		offset: 0,
	}, nil
}

// ReadAt io.ReaderAt interface method wrapper
func (b bufferedReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, fmt.Errorf("negative offset")
	}
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n = copy(p, b.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	return
}
