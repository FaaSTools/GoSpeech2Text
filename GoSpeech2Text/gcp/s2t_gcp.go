package aws

import (
	speech "cloud.google.com/go/speech/apiv2"
	speechpb "cloud.google.com/go/speech/apiv2/speechpb"
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	. "github.com/FaaSTools/GoText2Speech/GoSpeech2Text/shared"
	"io"
	"strings"
)

type S2TGoogleCloudPlatform struct {
	s2tClient *speech.Client
}

func (a S2TGoogleCloudPlatform) CreateServiceClient(credentials CredentialsHolder, region string) (S2TProvider, error) {
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		return a, err
	}
	a.s2tClient = client
	return a, nil
}

func (a S2TGoogleCloudPlatform) TransformOptions(text string, options SpeechToTextOptions) (string, SpeechToTextOptions, error) {
	return text, options, nil
}

// ExecuteS2T executes Speech-to-Text using AWS Transcribe service. The audio file on the given URL is transcribed into text
// using the given options. The created text is stored in the file specified at the destination parameter.
// The source string can either be an AWS S3 URI (starting with "s3://") or AWS S3 Object URL (starting with "https://").
// If an error occurs, returns empty string and error.
// If no error occurs, error return value is nil.
func (a S2TGoogleCloudPlatform) ExecuteS2T(sourceUrl string, destination string, options SpeechToTextOptions) error {
	r := <-a.ExecuteS2TDirect(sourceUrl, options)
	if r.Err != nil {
		return r.Err
	}

	// store file on destination
	storageClient, err3 := storage.NewClient(context.Background())
	if err3 != nil {
		fmt.Println(err3)
		return err3
	}

	obj := ParseGoogleUrl(destination)
	cloudObj := storageClient.Bucket(obj.Bucket).Object(obj.Key)
	wc := cloudObj.NewWriter(context.Background())

	outTextReader, strToReaderErr := StringToReader(r.Text)
	if strToReaderErr != nil {
		return strToReaderErr
	}

	if _, err4 := io.Copy(wc, outTextReader); err4 != nil {
		return fmt.Errorf("io.Copy: %w", err4)
	}
	if err5 := wc.Close(); err5 != nil {
		return fmt.Errorf("Writer.Close: %w", err5)
	}
	defer storageClient.Close()
	return nil
}

// ExecuteS2TDirect executes Speech-to-Text using AWS Transcribe service. The audio file on the given URL is transcribed into text
// using the given options. The created text is returned by this function.
// The source string can either be an AWS S3 URI (starting with "s3://") or AWS S3 Object URL (starting with "https://").
// If an error occurs, returns empty string and error.
// If no error occurs, error return value is nil.
func (a S2TGoogleCloudPlatform) ExecuteS2TDirect(sourceUrl string, options SpeechToTextOptions) <-chan S2TDirectResult {
	r := make(chan S2TDirectResult)

	go func() {
		defer close(r)

		req := &speechpb.RecognizeRequest{
			Recognizer: "",
			Config: &speechpb.RecognitionConfig{
				DecodingConfig: nil,
				Features: &speechpb.RecognitionFeatures{
					EnableSpokenEmojis:         options.EnableSpokenEmojis,
					EnableSpokenPunctuation:    options.EnableSpokenPunctuation,
					EnableAutomaticPunctuation: options.EnableAutomaticPunctuation,
					ProfanityFilter:            options.ProfanityFilter,
				},
				Adaptation: nil,
			},
			ConfigMask: nil,
		}

		if IsGoogleUrl(sourceUrl) {
			req.AudioSource = &speechpb.RecognizeRequest_Uri{
				Uri: sourceUrl,
			}
		} else {
			req.AudioSource = &speechpb.RecognizeRequest_Content{
				Content: nil,
			}
		}

		resp, err := a.s2tClient.Recognize(context.Background(), req)

		r <- S2TDirectResult{
			Text: StitchResultsTogether(resp),
			Err:  err,
		}
		return
	}()

	return r
}

func StitchResultsTogether(resp *speechpb.RecognizeResponse) string {
	resultText := ""
	for _, res := range resp.Results {
		var highestConfScore float32 = 0.0
		highestConfVal := ""
		for _, alt := range res.GetAlternatives() {
			if alt.GetConfidence() > highestConfScore {
				highestConfScore = alt.GetConfidence()
				highestConfVal = alt.GetTranscript()
			}
		}
		resultText += highestConfVal
	}
	return resultText
}

func (a S2TGoogleCloudPlatform) IsURLonOwnStorage(url string) bool {
	return IsGoogleUrl(url)
}

func (a S2TGoogleCloudPlatform) CloseServiceClient() error {
	if a.s2tClient == nil {
		fmt.Println("Warning: Couldn't close GCP S2T service client, because client doesn't exist.")
		return nil
	}
	return a.s2tClient.Close()
}

// gcpS2TsupportedFileTypes needs to be manually kept in-sync with GCP docs
// https://cloud.google.com/speech-to-text/docs/optimizing-audio-files-for-speech-to-text
var gcpS2TsupportedFileTypes = []string{
	"flac",
	"wav",
	"amr",
	"ogg",
}

func (a S2TGoogleCloudPlatform) SupportsFileType(fileType string) bool {
	for _, supFileType := range gcpS2TsupportedFileTypes {
		if strings.EqualFold(fileType, supFileType) {
			return true
		}
	}
	return false
}

func (a S2TGoogleCloudPlatform) SupportsDirectFileInput() bool {
	return true
}
