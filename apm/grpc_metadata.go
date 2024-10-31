package apm

import "google.golang.org/grpc/metadata"

const (
	metadataKeyPeerApp  = "peerApp"
	metadataKeyPeerHost = "peerHost"
)

// metadataSupplier is a supplier for the grpc metadata.
type metadataSupplier struct {
	metadata *metadata.MD
}

func (s *metadataSupplier) Get(key string) string {
	values := s.metadata.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (s *metadataSupplier) Set(key, value string) {
	s.metadata.Set(key, value)
}

func (s *metadataSupplier) Keys() []string {
	keys := make([]string, 0, len(*s.metadata))
	for key := range *s.metadata {
		keys = append(keys, key)
	}
	return keys
}

// getPeerInfo extracts the peer app and peer host from the metadata.
func getPeerInfo(md metadata.MD) (peerApp, peerHost string) {
	peerApps := md.Get(metadataKeyPeerApp)
	if len(peerApps) > 0 {
		peerApp = peerApps[0]
	}
	peerHosts := md.Get(metadataKeyPeerHost)
	if len(peerHosts) > 0 {
		peerHost = peerHosts[0]
	}
	return
}
