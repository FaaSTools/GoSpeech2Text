package main

import (
	"fmt"
	GoText2Speech "github.com/FaaSTools/GoText2Speech/GoSpeech2Text"
	"github.com/FaaSTools/GoText2Speech/GoSpeech2Text/providers"
	. "github.com/FaaSTools/GoText2Speech/GoSpeech2Text/shared"
	"os"
)

// main shows how S2T might be executed.
func main() {
	fmt.Println("Starting speech transcription...")

	s2tClient := GoText2Speech.CreateGoS2TClient(nil, "us-east-1")

	options := GetDefaultSpeechToTextOptions()
	options.LanguageConfig.LanguageCode = "en-US"
	options.Provider = providers.ProviderAWS

	bucket := "test"

	var err error = nil
	s2tClient, err = s2tClient.S2T("https://"+bucket+".s3.amazonaws.com/testfile.mp3", "https://"+bucket+".s3.amazonaws.com/testfile.txt", *options)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println("Speech successfully transcribed!")
}
