package GoText2Speech

import (
	"errors"
	"fmt"
	"github.com/FaaSTools/GoStorage/gostorage"
	s2t_aws "github.com/FaaSTools/GoText2Speech/GoSpeech2Text/aws"
	s2t_gcp "github.com/FaaSTools/GoText2Speech/GoSpeech2Text/gcp"
	"github.com/FaaSTools/GoText2Speech/GoSpeech2Text/providers"
	. "github.com/FaaSTools/GoText2Speech/GoSpeech2Text/shared"
	"github.com/aws/aws-sdk-go-v2/aws"
	"strings"
)

type GoS2TClient struct {
	providerInstances  map[providers.Provider]*S2TProvider
	region             string
	credentials        *CredentialsHolder
	redactedFileSuffix string
	DeleteTempFile     bool
	gostorageClient    *gostorage.GoStorage
}

func CreateGoS2TClient(credentials *CredentialsHolder, region string) GoS2TClient {
	if credentials == nil {
		awsCred, gcpCred := gostorage.LoadCredentialsFromDefaultLocation()
		awsCred = &aws.Credentials{
			AccessKeyID:     awsCred.AccessKeyID,
			SecretAccessKey: awsCred.SecretAccessKey,
		}
		credentials = &CredentialsHolder{
			AwsCredentials:    awsCred,
			GoogleCredentials: gcpCred,
		}
	}
	return GoS2TClient{
		providerInstances: make(map[providers.Provider]*S2TProvider),
		credentials:       credentials,
		region:            region,
		DeleteTempFile:    true,
	}
}

func (a GoS2TClient) getProviderInstance(provider providers.Provider) S2TProvider {
	if a.providerInstances[provider] == nil {
		prov := CreateProviderInstance(provider)
		a.providerInstances[provider] = &prov
	}
	return *a.providerInstances[provider]
}

func (a GoS2TClient) CloseProviderClient(provider providers.Provider) error {
	return a.getProviderInstance(provider).CloseServiceClient()
}

func (a GoS2TClient) CloseAllProviderClients() error {
	var allErrors error = nil
	for _, instance := range a.providerInstances {
		err := (*instance).CloseServiceClient()
		if err != nil {
			if allErrors == nil {
				allErrors = err
			} else {
				allErrors = errors.Join(allErrors, err)
			}
		}
	}
	return allErrors
}

// S2T Transforms the source file audio into text and stores the file in destination.
// The given source parameter specifies the location of the file. The file can have one of the following locations:
// * AWS S3
// * Google Cloud Storage
// * Other publicly accessible URL (beginning with 'http' or 'https')
// * Local file
// If the file is stored on some other provider, the file is uploaded to the storage service of the selected cloud provider.
// The service is automatically executed in the region in which the file is stored.
// If the given options specify a provider, this provider will be used.
// If the given options don't specify a provider, a provider will be chosen based on heuristics.
func (a GoS2TClient) S2T(source string, destination string, options SpeechToTextOptions) (GoS2TClient, error) {
	if options.Provider == providers.ProviderUnspecified {
		var err error
		options, err = a.determineProvider(options, source)
		if err != nil {
			return a, err
		}
	}

	fmt.Println("Provider: " + options.Provider)
	provider := a.getProviderInstance(options.Provider)

	err := provider.ExecuteS2T(source, destination, options)
	if err != nil {
		return a, err
	}

	return a, nil
}

type S2TDirectResultWrapper struct {
	Result S2TDirectResult
	Client GoS2TClient
}

func (a GoS2TClient) S2TDirect(source string, options SpeechToTextOptions) <-chan S2TDirectResultWrapper {
	r := make(chan S2TDirectResultWrapper)

	go func() {
		defer close(r)

		if options.Provider == providers.ProviderUnspecified {
			var err error
			options, err = a.determineProvider(options, source)
			if err != nil {
				r <- S2TDirectResultWrapper{
					Result: S2TDirectResult{
						Text: "",
						Err:  err,
					},
					Client: a,
				}
				return
			}
		}

		fmt.Println("Provider: " + options.Provider)
		provider := a.getProviderInstance(options.Provider)

		var transformOptionsErr error = nil
		source, options, transformOptionsErr = provider.TransformOptions(source, options)
		if transformOptionsErr != nil {
			r <- S2TDirectResultWrapper{
				Result: S2TDirectResult{
					Text: "",
					Err:  transformOptionsErr,
				},
				Client: a,
			}
			return
		}

		result := <-provider.ExecuteS2TDirect(source, options)
		r <- S2TDirectResultWrapper{
			Result: S2TDirectResult{
				Text: result.Text,
				Err:  result.Err,
			},
			Client: a,
		}
		return
	}()

	return r
}

// determineProvider executes heuristics in order to determine the most optimal cloud provider for speech transcription
// based on the input parameters.
// If returns the given SpeechToTextOptions with the 'Provider' property set to a specific provider.
func (a GoS2TClient) determineProvider(options SpeechToTextOptions, source string) (SpeechToTextOptions, error) {

	if strings.EqualFold(options.LanguageConfig.LanguageCode, "") {
		// no language specified -> language needs to be determined -> only available on AWS
		options.Provider = providers.ProviderAWS
		return options, nil
	}

	if !options.ContentRedactionConfig.IsEmpty() {
		// only available on AWS
		options.Provider = providers.ProviderAWS
		return options, nil
	}

	if options.ProfanityFilter || options.EnableAutomaticPunctuation || options.EnableSpokenPunctuation || options.EnableSpokenEmojis {
		// only available on GCP
		options.Provider = providers.ProviderGCP
		return options, nil
	}

	// use provider that supports the source file type
	fileType := GetFileTypeFromFileName(source)
	availProv := make(map[providers.Provider]bool)
	numAvailProv := 0
	for _, prov := range providers.GetAllProviders() {
		if a.getProviderInstance(prov).SupportsFileType(fileType) {
			availProv[prov] = true
			numAvailProv++
		} else {
			availProv[prov] = false
		}
	}

	// Only one provider supports file type -> return that provider
	if numAvailProv == 1 {
		for prov, avail := range availProv {
			if avail {
				options.Provider = prov
				return options, nil
			}
		}
	}

	// Either: requirement cannot be satisfied -> use any provider
	// Or: More than one provider supports file type -> use any provider
	options.Provider = DefaultProvider
	return options, nil
}

func (a GoS2TClient) initializeGoStorage() GoS2TClient {
	if a.gostorageClient == nil {
		a.gostorageClient = &gostorage.GoStorage{
			Credentials: *a.credentials,
		}
	}
	return a
}

func CreateProviderInstance(provider providers.Provider) S2TProvider {
	switch provider {
	case providers.ProviderAWS:
		return s2t_aws.S2TAmazonWebServices{}
	case providers.ProviderGCP:
		return s2t_gcp.S2TGoogleCloudPlatform{}
	default:
		return nil
	}
}

// IsProviderStorageUrl checks if the given string is a valid file URL for a storage service of one of the
// supported storage providers.
func (a GoS2TClient) IsProviderStorageUrl(url string) bool {
	for _, provider := range providers.GetAllProviders() {
		if a.getProviderInstance(provider).IsURLonOwnStorage(url) {
			return true
		}
	}
	return false
}
