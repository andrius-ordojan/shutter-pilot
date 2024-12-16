package media

type File interface {
	GetPath() string
	GetFingerprint() string
	SetFingerprint(fingerprint string)
	GetDestinationPath(base string) (string, error)
}

type mediaType string

const (
	photos mediaType = "photos"
	videos mediaType = "videos"
)
