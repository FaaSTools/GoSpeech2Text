package main

import (
	"context"
	GoSpeech2Text "github.com/FaaSTools/GoSpeech2Text/GoSpeech2Text"
	"github.com/aws/aws-lambda-go/lambda"
)

type MyEvent struct {
	Text         string `json:"text"`
	VoiceId      string `json:"voiceId"`
	TargetBucket string `json:"targetBucket"`
	TargetKey    string `json:"targetKey"`
}

func HandleRequest(ctx context.Context, ev MyEvent) error {
	ev = MyEvent{
		Text:         "Hello World",
		VoiceId:      "Joey",
		TargetBucket: "test",
		TargetKey:    "example01/example01-got2s.mp3",
	}

	s2tClient := GoSpeech2Text.CreateGoS2TClient(nil, "us-east-1")
	options := GoSpeech2Text.GetDefaultSpeechToTextOptions()
	options.Provider = GoSpeech2Text.providers.ProviderAWS
	var err error = nil
	target := "https://" + ev.TargetBucket + ".s3.amazonaws.com/" + ev.TargetKey
	s2tClient, err = s2tClient.S2T("https://test.s3.amazonaws.com/testfile.mp3", target, *options)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	lambda.Start(HandleRequest)

}
