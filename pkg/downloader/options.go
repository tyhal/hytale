package downloader

type Options func(*downloader)

// WithAllowAOTFail allows Ahead-of-Time (AOT) cache creation to fail without halting execution.
func WithAllowAOTFail() Options {
	return func(d *downloader) {
		d.allowAOTFail = true
	}
}

// WithCredentialsPath sets a custom credentials path for the downloader.
func WithCredentialsPath(path string) Options {
	return func(d *downloader) {
		d.credentialsPath = path
	}
}
