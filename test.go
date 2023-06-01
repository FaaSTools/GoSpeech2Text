package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	GoText2Speech "goTest/GoSpeech2Text"
	. "goTest/GoSpeech2Text/shared"
	"os"
)

// main shows how S2T might be executed.
func main() {
	fmt.Println("Starting speech transcription...")

	s2tClient := GoText2Speech.CreateGoS2TClient(CredentialsHolder{
		AwsCredentials: &session.Options{
			SharedConfigState: session.SharedConfigEnable,
			Profile:           "davidmeyer",
		},
	}, "us-east-1")

	// TODO
	options := GetDefaultSpeechToTextOptions()
	options.LanguageConfig.LanguageCode = "en"

	var err error = nil
	s2tClient, err = s2tClient.S2T("https://davemeyer-test.s3.amazonaws.com/testfile.mp3", "https://davemeyer-test.s3.amazonaws.com/S2T_Test_file_01.txt", *options)

	s2tClient.AwsTempBucket = "davemeyer-test"
	s2tClient, err = s2tClient.S2T("D:\\testfile.mp3", "D:\\S2T_Test_file_02.txt", *options)

	// TODO GCP example

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println("Speech successfully transcribed!")
}
