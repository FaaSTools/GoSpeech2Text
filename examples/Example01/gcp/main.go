package helloworld

import (
	"cloud.google.com/go/speech/apiv2/speechpb"
	"cloud.google.com/go/storage"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"io"
	"net/http"
)

// MyHTTPFunction Function is an HTTP handler
func MyHTTPFunction(w http.ResponseWriter, r *http.Request) {
	err := execS2T(r)
	if err != nil {
		fmt.Println("Error:", err.Error())
		_, err1 := io.WriteString(w, "Error: "+err.Error())
		if err1 != nil {
			fmt.Println("Error while writing error to output: ", err1)
		}
	} else {
		_, err1 := io.WriteString(w, "Done successfully!")
		if err1 != nil {
			fmt.Println("Error while writing success message to output: ", err1)
		}
	}
}

func execS2T(r *http.Request) error {
	//var ev = MyEvent{}
	var MyEvent struct { // don't count this struct
		Text         string `json:"Text"`
		VoiceId      string `json:"VoiceId"`
		TargetBucket string `json:"TargetBucket"`
		TargetKey    string `json:"TargetKey"`
	}
	MyEvent.TargetKey = "example01/example01-gcp.mp3"
	MyEvent.TargetBucket = "test"
	MyEvent.VoiceId = "en-US-News-N"
	MyEvent.Text = "Hello World"

	client, err := speech.NewClient(context.Background())
	if err != nil {
		return err
	}
	defer client.Close()
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
		ConfigMask:  nil,
		AudioSource: nil,
	}
	resp, err := client.Recognize(context.Background(), req)
	if err != nil {
		return err
	}
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
	csClient, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer csClient.Close()
	cloudObj := client.Bucket(bucket).Object(key)
	wc := cloudObj.NewWriter(ctx)
	if _, err = io.Copy(wc, resultText); err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %w", err)
	}
	return cloudObj.String()

	return nil
}

func init() {
	// Register an HTTP function with the Functions Framework
	functions.HTTP("MyHTTPFunction", MyHTTPFunction)
}

func main() {} // needs to be here, otherwise it can't be built
