package server

type ProtocolVersion string

const (
	V2025_03_26 ProtocolVersion = "2025-03-26"
	V2024_11_05 ProtocolVersion = "2024-11-05"

	// MINIMAL_FOR_STREAMABLE_HTTP is the minimal protocol version for
	// streamable http transport.
	//
	// see here: https://spec.modelcontextprotocol.io/specification/2025-03-26/changelog/
	MINIMAL_FOR_STREAMABLE_HTTP ProtocolVersion = V2025_03_26

	LATEST_PROTOCOL_VERSION = V2025_03_26
)

var (
	SUPPORTED_PROTOCOL_VERSIONS = []ProtocolVersion{
		LATEST_PROTOCOL_VERSION,
		V2024_11_05,
	}
)
