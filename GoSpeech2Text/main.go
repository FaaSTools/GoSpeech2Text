package GoText2Speech

import (
	"github.com/FaaSTools/GoStorage/gostorage"
	s2t_aws "goTest/GoSpeech2Text/aws"
	s2t_gcp "goTest/GoSpeech2Text/gcp"
	"goTest/GoSpeech2Text/providers"
	"goTest/GoSpeech2Text/shared"
)

type GoS2TClient struct {
	awsProvider          shared.S2TProvider
	gcpProvider          shared.S2TProvider
	region               string
	credentials          shared.CredentialsHolder
	AwsTempBucket        string
	GcpTempBucket        string
	redactedFileSuffix   string
	defaultFileExtension string
	DeleteTempFile       bool
	gostorageClient      gostorage.GoStorage
}

func CreateGoS2TClient(credentials shared.CredentialsHolder, region string) GoS2TClient {
	return GoS2TClient{
		awsProvider:    s2t_aws.S2TAmazonWebServices{},
		credentials:    credentials,
		region:         region,
		DeleteTempFile: true,
	}
}

func (a GoS2TClient) S2T(sourceUrl string, destination string, options shared.SpeechToTextOptions) (GoS2TClient, error) {

	return a, nil
}

func (a GoS2TClient) initializeGoStorage() GoS2TClient {
	// TODO check if exists
	a.gostorageClient = gostorage.GoStorage{} // TODO
	return a
}

func CreateProviderInstance(provider providers.Provider) shared.S2TProvider {
	switch provider {
	case providers.ProviderAWS:
		return s2t_aws.S2TAmazonWebServices{}
	case providers.ProviderGCP:
		return s2t_gcp.S2TGoogleCloudPlatform{}
	default:
		return nil
	}
}
