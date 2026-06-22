package protocol

const (
	// ContentTypeJSON обозначает JSON-содержимое.
	ContentTypeJSON = "application/json"

	// ContentTypeTextPlain обозначает текстовое содержимое без параметров кодировки.
	ContentTypeTextPlain = "text/plain"

	// ContentTypeTextPlainUTF8 обозначает текстовое содержимое в UTF-8.
	ContentTypeTextPlainUTF8 = "text/plain; charset=utf-8"

	// ContentTypeHTML обозначает HTML-содержимое без параметров кодировки.
	ContentTypeHTML = "text/html"

	// ContentTypeHTMLUTF8 обозначает HTML-содержимое в UTF-8.
	ContentTypeHTMLUTF8 = "text/html; charset=utf-8"

	// EncodingGzip обозначает gzip-сжатие HTTP-тела.
	EncodingGzip = "gzip"
)
