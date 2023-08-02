package aws

import (
	"context"
	"errors"
	"fmt"
	. "github.com/FaaSTools/GoText2Speech/GoSpeech2Text/shared"
	"github.com/aws/aws-sdk-go-v2/aws"
	transcribe "github.com/aws/aws-sdk-go-v2/service/transcribe"
	"github.com/aws/aws-sdk-go-v2/service/transcribe/types"
	"reflect"
	"strings"
	"time"
)

type S2TAmazonWebServices struct {
	credentials CredentialsHolder
	s2tClient   *transcribe.Client
	region      string
	//sess        client.ConfigProvider
}

type CredentialsProvider struct {
	credentials aws.Credentials
}

func (a S2TAmazonWebServices) GetDefaultRegion() string {
	return "us-east-1"
}

func (b CredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return b.credentials, nil
}

func (a S2TAmazonWebServices) CreateServiceClient(cred CredentialsHolder, region string) (S2TProvider, error) {
	credProv := CredentialsProvider{
		credentials: *cred.AwsCredentials,
	}
	a.credentials = cred
	a.region = region
	a.s2tClient = transcribe.New(transcribe.Options{
		Credentials: credProv,
		Region:      region,
	})
	return a, nil
}

func (a S2TAmazonWebServices) TransformOptions(text string, options SpeechToTextOptions) (string, SpeechToTextOptions, error) {
	return text, options, nil
}

func (a S2TAmazonWebServices) executeS2TInternal(sourceUrl string, destination string, options SpeechToTextOptions) (*transcribe.StartTranscriptionJobOutput, error) {
	jobName := options.TranscriptionJobName.GetTranscriptionJobName()

	contentRedaction := types.ContentRedaction{}

	if !reflect.DeepEqual(options.ContentRedactionConfig, ContentRedactionConfig{}) {
		contentRedaction.RedactionType = types.RedactionType(options.ContentRedactionConfig.ContentRedactionType)
		contentRedaction.RedactionOutput = types.RedactionOutput(options.ContentRedactionConfig.RedactionOutput)
		var entityTypes []types.PiiEntityType
		for _, ent := range options.ContentRedactionConfig.RedactionEntityTypes {
			entityTypes = append(entityTypes, types.PiiEntityType(*ent))
		}
		contentRedaction.PiiEntityTypes = entityTypes
	}

	identifyLanguage := false
	var languageCode = &options.LanguageConfig.LanguageCode
	// if language code is empty string: identify language automatically
	if strings.EqualFold("", *languageCode) {
		if !options.LanguageConfig.IdentifyMultipleLanguages {
			identifyLanguage = true
		}
	}

	bucket, key, destinationErr := GetBucketAndKeyFromAWSDestination(destination)
	if destinationErr != nil {
		return nil, errors.Join(errors.New(fmt.Sprintf("Couldn't run transcription because destination '%s' couldn't be parsed into AWS S3 bucket and key.", destination)), destinationErr)
	}

	var languageOptions []types.LanguageCode
	for _, l := range options.LanguageConfig.LanguageOptions {
		languageOptions = append(languageOptions, types.LanguageCode(*l))
	}

	awsContentRedaction := a.getAwsContentRedactionOptions(options)

	fileType := GetFileTypeFromFileName(sourceUrl)
	mediaFormat := getAwsFileType(fileType)

	jobInput := transcribe.StartTranscriptionJobInput{
		Media: &types.Media{
			MediaFileUri: &sourceUrl,
		},
		TranscriptionJobName:      &jobName,
		ContentRedaction:          awsContentRedaction,
		IdentifyLanguage:          &identifyLanguage,
		IdentifyMultipleLanguages: &options.LanguageConfig.IdentifyMultipleLanguages,
		LanguageCode:              types.LanguageCode(*languageCode),
		LanguageOptions:           languageOptions,
		MediaFormat:               mediaFormat,
		OutputBucketName:          &bucket,
		OutputKey:                 &key,
	}
	job, err := a.s2tClient.StartTranscriptionJob(context.Background(), &jobInput)

	if err != nil {
		errNew := errors.New("Error while starting transcription job: " + err.Error())
		fmt.Printf(errNew.Error())
		return job, errNew
	}

	return job, nil
}

func getTempDestination(sourceUrl string, options SpeechToTextOptions) string {
	return fmt.Sprintf("s3://%s/%d.%s", options.TempBucket, time.Now().UnixMilli(), options.DefaultTextFileExtension)
}

// ExecuteS2T executes Speech-to-Text using AWS Transcribe service. The audio file on the given URL is transcribed into text
// using the given options. The created text is stored in the file specified at the destination parameter.
// The source string can either be an AWS S3 URI (starting with "s3://") or AWS S3 Object URL (starting with "https://").
// If an error occurs, returns empty string and error.
// If no error occurs, error return value is nil.
func (a S2TAmazonWebServices) ExecuteS2T(sourceUrl string, destination string, options SpeechToTextOptions) error {
	_, err := a.executeS2TInternal(sourceUrl, destination, options)
	return err
}

// ExecuteS2TDirect executes Speech-to-Text using AWS Transcribe service. The audio file on the given URL is transcribed into text
// using the given options. The created text is returned by this function.
// The source string can either be an AWS S3 URI (starting with "s3://") or AWS S3 Object URL (starting with "https://").
// If an error occurs, returns empty string and error.
// If no error occurs, error return value is nil.
func (a S2TAmazonWebServices) ExecuteS2TDirect(sourceUrl string, options SpeechToTextOptions) <-chan S2TDirectResult {
	r := make(chan S2TDirectResult)

	go func() {
		defer close(r)

		originalJob, err := a.executeS2TInternal(sourceUrl, getTempDestination(sourceUrl, options), options)
		if err != nil {
			r <- S2TDirectResult{
				Text: "",
				Err:  err,
			}
			return
		}

		jobName := originalJob.TranscriptionJob.TranscriptionJobName
		jobStatus := originalJob.TranscriptionJob.TranscriptionJobStatus
		var err2 error = nil
		var job *transcribe.GetTranscriptionJobOutput = nil
		now := time.Now()
		lastCheckTime := now.UnixMilli()
		for jobStatus != "COMPLETED" {
			if jobStatus == "FAILED" {
				r <- S2TDirectResult{
					Text: "",
					Err:  errors.New(fmt.Sprintf("Error occurred during transcription: %s\n", *job.TranscriptionJob.FailureReason)),
				}
				return
			}

			now = time.Now()
			if (now.UnixMilli() - lastCheckTime) > options.TranscriptionJobCheckIntervalMs {
				//fmt.Printf("Check job status of %s\n", *jobName)
				job, err2 = a.s2tClient.GetTranscriptionJob(context.Background(), &transcribe.GetTranscriptionJobInput{TranscriptionJobName: jobName})
				if err2 != nil {
					r <- S2TDirectResult{
						Text: "",
						Err:  err2,
					}
					return
				}
				jobStatus = job.TranscriptionJob.TranscriptionJobStatus
			}
		}
		fmt.Printf("job done\n")
	}()

	return r
}

// getAwsContentRedactionOptions converts the abstracted GoSpeech2Text content redaction options to AWS content redaction options.
func (a S2TAmazonWebServices) getAwsContentRedactionOptions(options SpeechToTextOptions) *types.ContentRedaction {
	var awsContentRedaction *types.ContentRedaction = nil
	if !options.ContentRedactionConfig.IsEmpty() {
		awsContentRedaction = &types.ContentRedaction{
			RedactionOutput: types.RedactionOutput(options.ContentRedactionConfig.RedactionOutput),
			RedactionType:   types.RedactionType(options.ContentRedactionConfig.ContentRedactionType),
		}
		var piiEntityTypes []types.PiiEntityType = nil
		for _, entityType := range options.ContentRedactionConfig.RedactionEntityTypes {
			piiEntityTypes = append(piiEntityTypes, types.PiiEntityType(*entityType))
		}
		awsContentRedaction.PiiEntityTypes = piiEntityTypes
	}
	return awsContentRedaction
}

// GetBucketAndKeyFromAWSDestination receives either an AWS S3 URI (starting with "s3://") or
// AWS S3 Object URL (starting with "https://") and returns the bucket and key (without preceding slash) of the file.
// If the given destination is not valid, then two empty strings and an error is returned.
// copied from GoText2Speech
func GetBucketAndKeyFromAWSDestination(destination string) (string, string, error) {
	if strings.HasPrefix(destination, "s3://") {
		withoutPrefix, _ := strings.CutPrefix(destination, "s3://")
		bucket := strings.Split(withoutPrefix, "/")[0]
		key, _ := strings.CutPrefix(withoutPrefix, bucket+"/")
		return bucket, key, nil
	} else if strings.HasPrefix(destination, "https://") && strings.Contains(destination, "s3") {
		withoutPrefix, _ := strings.CutPrefix(destination, "https://")
		dotSplits := strings.SplitN(withoutPrefix, ".", 3)
		bucket := dotSplits[0]
		key := strings.SplitN(dotSplits[2], "/", 2)[1]
		return bucket, key, nil
	} else {
		return "", "", errors.New(fmt.Sprintf("The given destination '%s' is not a valid S3 URI or S3 Object URL.", destination))
	}
}

func (a S2TAmazonWebServices) IsURLonOwnStorage(url string) bool {
	return IsAWSUrl(url)
}

func (a S2TAmazonWebServices) CloseServiceClient() error {
	// AWS clients cannot be closed
	return nil
}

func (a S2TAmazonWebServices) SupportsFileType(fileType string) bool {
	values := types.MediaFormatMp3.Values()
	for _, supFileType := range values {
		if strings.EqualFold(fileType, string(supFileType)) {
			return true
		}
	}
	return false
}

// getAwsFileType turns the given fileType string into an AWS media format type.
// The given fileType must not start with a period.
func getAwsFileType(fileType string) types.MediaFormat {
	return types.MediaFormat(fileType)
}

func (a S2TAmazonWebServices) SupportsDirectFileInput() bool {
	return false
}

func (a S2TAmazonWebServices) GetStorageUrl(region string, bucket string, key string) string {
	return "https://" + bucket + ".s3." + region + ".amazonaws.com/" + key
}
