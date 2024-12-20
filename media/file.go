package media

import "sync"

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

type LazyPath struct {
	err  error
	path string
	once sync.Once
}

func (lp *LazyPath) GetDestinationPath(compute func() (string, error)) (string, error) {
	lp.once.Do(func() {
		lp.path, lp.err = compute()
	})
	return lp.path, lp.err
}
