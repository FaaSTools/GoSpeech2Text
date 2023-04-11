package main

import (
	transcribe "github.com/aws/aws-sdk-go/service/transcribeservice"
	. "goTest/GoSpeech2Text/shared"
)

// main shows how T2S might be executed.
func main() {
	//fmt.Println("Starting speech transcription...")

	// TODO
	options := SpeechToTextOptions{
		ContentRedactionConfig: ContentRedactionConfig{
			// Either use value specified by the GoSpeech2Text abstraction
			//ContentRedactionType: RedactionTypePersonallyIdentifiableInformation,
			// Or use original value from AWS Go SDK
			ContentRedactionType: transcribe.RedactionTypePii,
		},
	}

	//fmt.Println("Speech successfully transcribed!")
}
