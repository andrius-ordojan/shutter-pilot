package media

import "sync"

type File interface {
	GetPath() string
	GetFingerprint() string
	SetFingerprint(fingerprint string)
	GetDestinationPath(base string) (string, error)
}

type (
	MediaType string
	mediaLoc  string
)

const (
	JpgMedia MediaType = "jpg"
	RafMedia MediaType = "raf"
	MovMedia MediaType = "mov"
	photos   mediaLoc  = "photos"
	videos   mediaLoc  = "videos"
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
