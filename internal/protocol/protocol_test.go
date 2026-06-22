package protocol

import "testing"

func TestTransportConstants(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "hash header", got: HeaderHashSHA256, want: "HashSHA256"},
		{name: "real IP header", got: HeaderRealIP, want: "X-Real-IP"},
		{name: "content type header", got: HeaderContentType, want: "Content-Type"},
		{name: "content encoding header", got: HeaderContentEncoding, want: "Content-Encoding"},
		{name: "accept encoding header", got: HeaderAcceptEncoding, want: "Accept-Encoding"},
		{name: "content encryption header", got: HeaderContentEncryption, want: "Content-Encryption"},
		{name: "real IP metadata", got: MetadataRealIP, want: "x-real-ip"},
		{name: "JSON content type", got: ContentTypeJSON, want: "application/json"},
		{name: "plain text content type", got: ContentTypeTextPlain, want: "text/plain"},
		{name: "plain text UTF-8 content type", got: ContentTypeTextPlainUTF8, want: "text/plain; charset=utf-8"},
		{name: "HTML content type", got: ContentTypeHTML, want: "text/html"},
		{name: "HTML UTF-8 content type", got: ContentTypeHTMLUTF8, want: "text/html; charset=utf-8"},
		{name: "gzip encoding", got: EncodingGzip, want: "gzip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}
