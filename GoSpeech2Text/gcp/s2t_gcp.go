package aws

import (
	"fmt"
	"goTest/GoSpeech2Text/shared"
)

type S2TGoogleCloudPlatform struct {
	s2tClient any // TODO set correct type
}

func (a S2TGoogleCloudPlatform) CreateServiceClient(credentials shared.CredentialsHolder, region string) shared.S2TProvider {
	fmt.Println("Not yet implemented")
	// TODO implement
	return a
}

func (a S2TGoogleCloudPlatform) TransformOptions(text string, options shared.SpeechToTextOptions) (string, shared.SpeechToTextOptions, error) {
	fmt.Println("Not yet implemented")
	return text, options, nil
}

// ExecuteS2T executes Speech-to-Text using AWS Transcribe service. The audio file on the given URL is transcribed into text
// using the given options. The created text is stored in the file specified at the destination parameter.
// The source string can either be an AWS S3 URI (starting with "s3://") or AWS S3 Object URL (starting with "https://").
// If an error occurs, returns empty string and error.
// If no error occurs, error return value is nil.
func (a S2TGoogleCloudPlatform) ExecuteS2T(sourceUrl string, destination string, options shared.SpeechToTextOptions) error {
	fmt.Println("Not yet implemented")
	// TODO implement
	return nil
}

// ExecuteS2TDirect executes Speech-to-Text using AWS Transcribe service. The audio file on the given URL is transcribed into text
// using the given options. The created text is returned by this function.
// The source string can either be an AWS S3 URI (starting with "s3://") or AWS S3 Object URL (starting with "https://").
// If an error occurs, returns empty string and error.
// If no error occurs, error return value is nil.
func (a S2TGoogleCloudPlatform) ExecuteS2TDirect(sourceUrl string, options shared.SpeechToTextOptions) (string, error) {
	fmt.Println("Not yet implemented")
	// TODO execute ExecuteS2T and return file contents.
	return "", nil
}
