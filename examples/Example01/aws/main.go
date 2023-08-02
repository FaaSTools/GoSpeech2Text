package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/transcribe"
	"github.com/aws/aws-sdk-go-v2/service/transcribe/types"
	"strconv"
	"time"
)

type MyEvent struct {
	SourceBucket string `json:"sourceBucket"`
	SourceKey    string `json:"sourceKey"`
	TargetBucket string `json:"targetBucket"`
	TargetKey    string `json:"targetKey"`
}

func HandleRequest(ctx context.Context, ev0 MyEvent) error {
	ev := MyEvent{
		SourceBucket: "test",
		SourceKey:    "example01/example01-aws.mp3",
		TargetBucket: "test",
		TargetKey:    "example01/example01-aws.txt",
	}

	cfg, err0 := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err0 != nil {
		return err0
	}
	jobName := "example01-" + strconv.FormatInt(time.Now().UnixMilli(), 10)
	identifyLanguage := true
	sourceUrl := "s3://" + ev.SourceBucket + "/" + ev.TargetBucket
	client := transcribe.NewFromConfig(cfg)
	jobInput := &transcribe.StartTranscriptionJobInput{
		Media: &types.Media{
			MediaFileUri: &sourceUrl,
		},
		TranscriptionJobName: &jobName,
		ContentRedaction:     nil,
		IdentifyLanguage:     &identifyLanguage,
		LanguageCode:         "",
		MediaFormat:          types.MediaFormatMp3,
		OutputBucketName:     &ev.TargetBucket,
		OutputKey:            &ev.TargetKey,
	}
	_, err := client.StartTranscriptionJob(context.Background(), jobInput)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
