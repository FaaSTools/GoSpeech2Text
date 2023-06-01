package aws

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/transcribeservice"
	ContentRedaction "github.com/aws/aws-sdk-go/service/transcribeservice"
	. "goTest/GoSpeech2Text/shared"
	"reflect"
	"strings"
	"unsafe"
)

type S2TAmazonWebServices struct {
	s2tClient   *transcribeservice.TranscribeService
	credentials CredentialsHolder
	sess        client.ConfigProvider
}

func (a S2TAmazonWebServices) CreateServiceClient(credentials CredentialsHolder, region string) S2TProvider {
	credentials.AwsCredentials.Config.Region = &region
	sess := session.Must(session.NewSessionWithOptions(*credentials.AwsCredentials))
	a.sess = sess
	a.s2tClient = transcribeservice.New(sess)
	return a
}

func (a S2TAmazonWebServices) TransformOptions(text string, options SpeechToTextOptions) (string, SpeechToTextOptions, error) {
	return text, options, nil
}

// ExecuteS2T executes Speech-to-Text using AWS Transcribe service. The audio file on the given URL is transcribed into text
// using the given options. The created text is stored in the file specified at the destination parameter.
// The source string can either be an AWS S3 URI (starting with "s3://") or AWS S3 Object URL (starting with "https://").
// If an error occurs, returns empty string and error.
// If no error occurs, error return value is nil.
func (a S2TAmazonWebServices) ExecuteS2T(sourceUrl string, destination string, options SpeechToTextOptions) error {
	jobName := options.TranscriptionJobName.GetTranscriptionJobName()

	contentRedaction := ContentRedaction.ContentRedaction{}

	if !reflect.DeepEqual(options.ContentRedactionConfig, ContentRedactionConfig{}) {
		contentRedaction.RedactionType = (*string)(&options.ContentRedactionConfig.ContentRedactionType)
		contentRedaction.RedactionOutput = (*string)(&options.ContentRedactionConfig.RedactionOutput)
		contentRedaction.PiiEntityTypes = *(*[]*string)(unsafe.Pointer(&options.ContentRedactionConfig.RedactionEntityTypes))
	}

	identifyLanguage := false
	var languageCode *string = nil
	if strings.EqualFold("", options.LanguageConfig.LanguageCode) {
		languageCode = &options.LanguageConfig.LanguageCode
		if !options.LanguageConfig.IdentifyMultipleLanguages {
			identifyLanguage = true
		}
	}

	bucket, key, destinationErr := GetBucketAndKeyFromAWSDestination(destination)
	if destinationErr != nil {
		return errors.Join(errors.New(fmt.Sprintf("Couldn't run transcription because destination '%s' couldn't be parsed into AWS S3 bucket and key.", destination)), destinationErr)
	}

	media := transcribeservice.Media{
		MediaFileUri: &destination,
		// TODO redaction
	}

	// TODO other options
	job, err := a.s2tClient.StartTranscriptionJob(&transcribeservice.StartTranscriptionJobInput{
		ContentRedaction:          &contentRedaction,
		IdentifyLanguage:          &identifyLanguage,
		IdentifyMultipleLanguages: &options.LanguageConfig.IdentifyMultipleLanguages,
		JobExecutionSettings:      nil,
		KMSEncryptionContext:      nil,
		LanguageCode:              languageCode,
		LanguageIdSettings:        nil,
		LanguageOptions:           options.LanguageConfig.LanguageOptions,
		Media:                     &media,
		MediaFormat:               nil,
		MediaSampleRateHertz:      nil,
		ModelSettings:             nil,
		OutputBucketName:          &bucket,
		OutputKey:                 &key,
		TranscriptionJobName:      &jobName,
	})
	fmt.Printf("%s", job)

	if err != nil {
		errNew := errors.New("Error while starting transcription job: " + err.Error())
		fmt.Printf(errNew.Error())
		return errNew
	}

	return nil
}

// ExecuteS2TDirect executes Speech-to-Text using AWS Transcribe service. The audio file on the given URL is transcribed into text
// using the given options. The created text is returned by this function.
// The source string can either be an AWS S3 URI (starting with "s3://") or AWS S3 Object URL (starting with "https://").
// If an error occurs, returns empty string and error.
// If no error occurs, error return value is nil.
func (a S2TAmazonWebServices) ExecuteS2TDirect(sourceUrl string, options SpeechToTextOptions) (string, error) {
	// TODO execute ExecuteS2T and return file contents.
	return "", nil
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
