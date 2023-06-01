package shared

type S2TProvider interface {
	// TransformOptions Transforms the given options object such that it can be used for the chosen provider.
	TransformOptions(text string, options SpeechToTextOptions) (string, SpeechToTextOptions, error)
	// CreateServiceClient creates s2t client for the chosen provider and stores it in the struct.
	CreateServiceClient(credentials CredentialsHolder, region string) S2TProvider // TODO parameters?
	ExecuteS2TDirect(sourceUrl string, options SpeechToTextOptions) (string, error)
	ExecuteS2T(source string, destination string, options SpeechToTextOptions) error
	// TODO close function for client?
}
