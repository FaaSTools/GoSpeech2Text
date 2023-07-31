package shared

type S2TDirectResult struct {
	Text string
	Err  error
}

type S2TProvider interface {
	// TransformOptions Transforms the given options object such that it can be used for the chosen provider.
	TransformOptions(sourceUrl string, options SpeechToTextOptions) (string, SpeechToTextOptions, error)
	// CreateServiceClient creates s2t client for the chosen provider and stores it in the struct.
	CreateServiceClient(credentials CredentialsHolder, region string) (S2TProvider, error)
	ExecuteS2TDirect(sourceUrl string, options SpeechToTextOptions) <-chan S2TDirectResult
	ExecuteS2T(source string, destination string, options SpeechToTextOptions) error
	// IsURLonOwnStorage checks if the given URL references a file that is hosted on the provider's own storage service
	// (i.e. S3 on AWS or Cloud Storage on GCP).
	IsURLonOwnStorage(url string) bool
	// CloseServiceClient closes the connection of the s2t client in the struct (if such an operation is available on the provider).
	CloseServiceClient() error
	// SupportsFileType returns true if the given file type is supported for the Speech-to-Text service of the provider.
	// File type needs to be specified without preceding period.
	SupportsFileType(fileType string) bool
	SupportsDirectFileInput() bool
}
