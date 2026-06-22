package protocol

const (
	// HeaderHashSHA256 содержит имя заголовка с подписью HTTP-тела.
	HeaderHashSHA256 = "HashSHA256"

	// HeaderRealIP содержит имя HTTP-заголовка с IP-адресом Агента.
	HeaderRealIP = "X-Real-IP"

	// HeaderContentType содержит имя заголовка с типом содержимого.
	HeaderContentType = "Content-Type"

	// HeaderContentEncoding содержит имя заголовка с кодированием тела сообщения.
	HeaderContentEncoding = "Content-Encoding"

	// HeaderAcceptEncoding содержит имя заголовка с поддерживаемыми кодировками ответа.
	HeaderAcceptEncoding = "Accept-Encoding"

	// HeaderContentEncryption содержит имя заголовка со схемой шифрования тела запроса.
	HeaderContentEncryption = "Content-Encryption"
)
