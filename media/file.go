package media

type File interface {
	GetPath() string
	GetFingerprint() string
	SetFingerprint(fingerprint string)
	GetDestinationPath(base string) (string, error)
}

type Type string

const (
	Photos Type = "photos"
	Videos Type = "videos"
)
