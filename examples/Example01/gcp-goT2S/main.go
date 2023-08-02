package helloworld

import (
	"context"
	"fmt"
	GoText2Speech "github.com/FaaSTools/GoText2Speech/GoSpeech2Text"
	"github.com/FaaSTools/GoText2Speech/GoSpeech2Text/providers"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"golang.org/x/oauth2/google"
	"io"
	"net/http"
)

func execT2S(r *http.Request) error {
	var MyEvent struct { // don't count this struct
		Text         string `json:"Text"`
		VoiceId      string `json:"VoiceId"`
		TargetBucket string `json:"TargetBucket"`
		TargetKey    string `json:"TargetKey"`
	}
	MyEvent.TargetKey = "example01/example01-got2s.mp3"
	MyEvent.TargetBucket = "test"
	MyEvent.VoiceId = "en-US-News-N"
	MyEvent.Text = "Hello World"

	region := "us-east-1"
	googleCredentials, err := google.CredentialsFromJSON(
		context.Background(),
		[]byte("CREDENTIALS_HERE"),
		"https://www.googleapis.com/auth/devstorage.full_control",
		"https://www.googleapis.com/auth/cloud-platform",
	)
	fmt.Println("err while reading credentials:", err)

	cred := &goS2TShared.CredentialsHolder{
		GoogleCredentials: googleCredentials,
	}

	s2tClient := GoText2Speech.CreateGoS2TClient(nil, "us-east-1")
	options := GetDefaultSpeechToTextOptions()
	options.LanguageConfig.LanguageCode = "en"
	options.Provider = providers.ProviderGCP
	var err2 error = nil
	target := "https://" + bucket + ".s3.amazonaws.com/" + key
	s2tClient, err2 = s2tClient.S2T("https://test.s3.amazonaws.com/testfile.mp3", target, *options)
	if err2 != nil {
		return err2
	}
	return nil
}

func init() {
	// Register an HTTP function with the Functions Framework
	functions.HTTP("MyHTTPFunction", MyHTTPFunction)
}

func main() {} // needs to be here, otherwise it can't be built

// MyHTTPFunction Function is an HTTP handler
func MyHTTPFunction(w http.ResponseWriter, r *http.Request) {
	err := execT2S(r)
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
