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
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type GoS2TClient struct {
	providerInstances  map[providers.Provider]*S2TProvider
	region             *string
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

	var regionPtr *string = nil
	if !strings.EqualFold(region, "") {
		regionPtr = &region
	}

	s2tClient := GoS2TClient{
		providerInstances: make(map[providers.Provider]*S2TProvider),
		credentials:       credentials,
		region:            regionPtr,
		DeleteTempFile:    true,
	}
	s2tClient = s2tClient.initializeGoStorage()
	return s2tClient
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

	newSource := source
	// in some cases, a temporary file needs to be uploaded to a storage service
	var tmpUploadedFile *gostorage.GoStorageObject = nil

	provider := a.getProviderInstance(options.Provider)
	if a.IsProviderStorageUrl(source) {
		storageObj := ParseUrlToGoStorageObject(source)
		if a.region == nil {
			// Region preference is not set -> use region of source file
			region := storageObj.Region
			if strings.EqualFold(region, "") {
				region = provider.GetDefaultRegion()
			}
			a.region = &region
			cred := a.credentials

			var errServiceClient error = nil
			provider, errServiceClient = provider.CreateServiceClient(*cred, region)
			if errServiceClient != nil {
				return a, errors.Join(errors.New("error while creating S2T service client"), errServiceClient)
			}

		} else if !strings.EqualFold(*a.region, storageObj.Region) {
			// File is in different region -> move file
			destStorageObj := gostorage.GoStorageObject{
				Bucket:        storageObj.Bucket,
				Key:           storageObj.Key,
				Region:        *a.region,
				IsLocal:       false,
				LocalFilePath: "",
				ProviderType:  storageObj.ProviderType,
			}
			a.gostorageClient.Copy(storageObj, destStorageObj)
		}
	} else if strings.HasPrefix(source, "http") { // file somewhere else online
		if a.region == nil {
			// Region preference is not set and source doesn't have region -> use region of destination file (if exists)
			storageObjDest := ParseUrlToGoStorageObject(destination)
			region := storageObjDest.Region
			if strings.EqualFold(region, "") {
				region = provider.GetDefaultRegion()
			}
			a.region = &region
		}

		if !provider.SupportsDirectFileInput() {
			// direct file input not supported -> download file and upload to storage service

			storageObj, errExtUrlToStorageObj := a.externalUrlToStorageObj(source, options)
			if errExtUrlToStorageObj != nil {
				return a, errExtUrlToStorageObj
			}

			a.gostorageClient.UploadFile(*storageObj)
			newSource = provider.GetStorageUrl(storageObj.Region, storageObj.Bucket, storageObj.Key)
			tmpUploadedFile = storageObj

			// delete temporarily stored audio file (if needed).
			// The temporarily uploaded file is deleted after S2T has been executed.
			if a.DeleteTempFile {
				removeErr := os.Remove(storageObj.LocalFilePath)
				if removeErr != nil {
					return a, removeErr
				}
			}
		}

	} else { // local file
		if a.region == nil {
			// Region preference is not set and source doesn't have region -> use region of destination file (if exists)
			storageObjDest := ParseUrlToGoStorageObject(destination)
			region := storageObjDest.Region
			if strings.EqualFold(region, "") {
				region = provider.GetDefaultRegion()
			}
			a.region = &region
		}

		if !provider.SupportsDirectFileInput() {
			// direct file input not supported -> upload to storage

			storageObj := gostorage.GoStorageObject{
				Bucket:        options.TempBucket,
				Key:           strconv.FormatInt(time.Now().UnixNano(), 10), // essentially random key
				Region:        *a.region,
				IsLocal:       true,
				LocalFilePath: source,
				ProviderType:  ProviderToGoStorageProvider(options.Provider),
			}
			a.gostorageClient.UploadFile(storageObj)
			newSource = provider.GetStorageUrl(storageObj.Region, storageObj.Bucket, storageObj.Key)
			tmpUploadedFile = &storageObj
		}
	}

	err := provider.ExecuteS2T(newSource, destination, options)
	if err != nil {
		return a, err
	}

	// Delete temporarily uploaded file (if it should be deleted and if it exists)
	if a.DeleteTempFile && (tmpUploadedFile != nil) {
		a.gostorageClient.DeleteFile(*tmpUploadedFile)
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

		newSource := source
		// in some cases, a temporary file needs to be uploaded to a storage service
		var tmpUploadedFile *gostorage.GoStorageObject = nil

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

		provider := a.getProviderInstance(options.Provider)
		if a.IsProviderStorageUrl(source) {
			storageObj := ParseUrlToGoStorageObject(source)
			if a.region == nil {
				// Region preference is not set -> use region of source file
				region := storageObj.Region
				a.region = &region
				cred := a.credentials

				var errServiceClient error = nil
				provider, errServiceClient = provider.CreateServiceClient(*cred, region)
				if errServiceClient != nil {
					r <- S2TDirectResultWrapper{
						Result: S2TDirectResult{
							Text: "",
							Err:  errors.Join(errors.New("error while creating S2T service client"), errServiceClient),
						},
						Client: a,
					}
					return
				}

			} else if !strings.EqualFold(*a.region, storageObj.Region) {
				// File is in different region -> move file
				destStorageObj := gostorage.GoStorageObject{
					Bucket:        storageObj.Bucket,
					Key:           storageObj.Key,
					Region:        *a.region,
					IsLocal:       false,
					LocalFilePath: "",
					ProviderType:  storageObj.ProviderType,
				}
				a.gostorageClient.Copy(storageObj, destStorageObj)
			}
		} else if strings.HasPrefix(source, "http") { // file somewhere else online
			if a.region == nil {
				region := provider.GetDefaultRegion()
				a.region = &region
			}

			if !provider.SupportsDirectFileInput() {
				// direct file input not supported -> download file and upload to storage service

				storageObj, errExtUrlToStorageObj := a.externalUrlToStorageObj(source, options)
				if errExtUrlToStorageObj != nil {
					r <- S2TDirectResultWrapper{
						Result: S2TDirectResult{
							Text: "",
							Err:  errExtUrlToStorageObj,
						},
						Client: a,
					}
					return
				}

				a.gostorageClient.UploadFile(*storageObj)
				newSource = provider.GetStorageUrl(storageObj.Region, storageObj.Bucket, storageObj.Key)
				tmpUploadedFile = storageObj

				// delete temporarily stored audio file (if needed).
				// The temporarily uploaded file is deleted after S2T has been executed.
				if a.DeleteTempFile {
					removeErr := os.Remove(storageObj.LocalFilePath)
					if removeErr != nil {
						r <- S2TDirectResultWrapper{
							Result: S2TDirectResult{
								Text: "",
								Err:  errors.Join(errors.New("error while removing temporarily stored audio file"), removeErr),
							},
							Client: a,
						}
						return
					}
				}
			}

		} else { // local file
			if a.region == nil {
				region := provider.GetDefaultRegion()
				a.region = &region
			}

			if !provider.SupportsDirectFileInput() {
				// direct file input not supported -> upload to storage

				storageObj := gostorage.GoStorageObject{
					Bucket:        options.TempBucket,
					Key:           strconv.FormatInt(time.Now().UnixNano(), 10), // essentially random key
					Region:        *a.region,
					IsLocal:       true,
					LocalFilePath: source,
					ProviderType:  ProviderToGoStorageProvider(options.Provider),
				}
				a.gostorageClient.UploadFile(storageObj)
				newSource = provider.GetStorageUrl(storageObj.Region, storageObj.Bucket, storageObj.Key)
				tmpUploadedFile = &storageObj
			}
		}

		var transformOptionsErr error = nil
		newSource, options, transformOptionsErr = provider.TransformOptions(newSource, options)
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

		result := <-provider.ExecuteS2TDirect(newSource, options)

		// Delete temporarily uploaded file (if it should be deleted and if it exists)
		if a.DeleteTempFile && (tmpUploadedFile != nil) {
			a.gostorageClient.DeleteFile(*tmpUploadedFile)
		}

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

func (a GoS2TClient) externalUrlToStorageObj(url string, options SpeechToTextOptions) (*gostorage.GoStorageObject, error) {
	reader, errDownload := ReadFromUrl(url)
	if errDownload != nil {
		return nil, errDownload
	}

	// close reader after function call ended
	defer func(Reader io.ReadCloser) {
		errClose := Reader.Close()
		if errClose != nil {
			fmt.Printf(errors.Join(errors.New(fmt.Sprintf("A non-fatal error occurred while closing the HTTP response for the source file '%s'.", url)), errClose).Error())
		}
	}(reader)

	tmpFile, errTmpFile := os.CreateTemp("", "sample")
	if errTmpFile != nil {
		return nil, errTmpFile
	}

	errStoreFile := StoreAudioToLocalFile(reader, tmpFile)
	if errStoreFile != nil {
		return nil, errStoreFile
	}

	errClose := tmpFile.Close()
	if errClose != nil {
		return nil, errClose
	}

	storageObj := gostorage.GoStorageObject{
		Bucket:        options.TempBucket,
		Key:           strconv.FormatInt(time.Now().UnixNano(), 10), // essentially random key
		Region:        *a.region,
		IsLocal:       true,
		LocalFilePath: tmpFile.Name(),
		ProviderType:  ProviderToGoStorageProvider(options.Provider),
	}

	return &storageObj, nil
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
