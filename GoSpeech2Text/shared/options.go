package shared

import (
	"goTest/GoSpeech2Text/providers"
	"strconv"
	"time"
)

type SpeechToTextOptions struct {
	_        struct{}
	Provider providers.Provider
	// TranscriptionJobName specifies a configuration for creating unique transcription job names.
	// On AWS, every transcription job needs a unique name. This name must be unique within an AWS account.
	// This property is ignored on GCP.
	// See docs for TranscriptionJobNameConfig for more info.
	TranscriptionJobName TranscriptionJobNameConfig
	// ContentRedactionConfig is currently only available on AWS.
	// If nil, content redaction is deactivated.
	// See AWS docs: https://docs.aws.amazon.com/sdk-for-go/api/service/transcribeservice/#ContentRedaction
	ContentRedactionConfig ContentRedactionConfig
	// EnableAutomaticPunctuation is currently only available on GCP.
	// If 'true', adds punctuation to resulting text.
	// This feature is only available on select languages.
	// If it is enabled on other languages, it has no effect at all.
	EnableAutomaticPunctuation bool
	// EnableSpokenPunctuation is currently only available on GCP.
	// If 'true', spoken punctuation is replaced by the punctuation symbol.
	// Example: "how are you question mark" becomes "how are you?"
	// See GCP docs: https://cloud.google.com/speech-to-text/docs/spoken-punctuation
	EnableSpokenPunctuation bool
	// EnableSpokenEmojis is currently only available on GCP.
	// From GCP docs:
	// If 'true', adds spoken emoji formatting for the request. This will replace spoken emojis
	// with the corresponding Unicode symbols in the final transcript.
	// See GCP docs: https://cloud.google.com/speech-to-text/docs/spoken-emoji
	EnableSpokenEmojis bool
	// ProfanityFilter is currently only available on GCP.
	// From GCP docs:
	// If set to 'true', the server will attempt to filter out
	// profanities, replacing all but the initial character in each filtered word
	// with asterisks, e.g. "f***". If set to `false` or omitted, profanities won't be filtered out.
	// See GCP docs: https://pkg.go.dev/cloud.google.com/go/speech@v1.15.0/apiv1/speechpb#RecognitionConfig
	ProfanityFilter bool
}

// ContentRedactionConfig Configuration for content redaction.
// This struct is an abstraction for the ContentRedaction struct in AWS Go SDK
// (and for a possible configuration struct for GCP, in the future).
// Currently only available on AWS.
// See AWS docs: https://docs.aws.amazon.com/sdk-for-go/api/service/transcribeservice/#ContentRedaction
type ContentRedactionConfig struct {
	_ struct{}
	// ContentRedactionType specified the category of information that should be redacted.
	// In case a new valid content redaction type is available on AWS, you can also specify a string and circumvent
	// the abstracted type specified by GoSpeech2Text.
	//
	// If undefined (i.e. empty string ""), RedactionTypePersonallyIdentifiableInformation is chosen.
	// Since this is currently the only supported value, ContentRedactionType can be left undefined.
	ContentRedactionType ContentRedactionType
	// RedactionEntityTypes specified which types of information should be redacted in the transcript.
	// Available values are specified in the RedactionEntityType enum.
	// In case a new valid content redaction type is available on AWS, you can also specify a string and circumvent
	// the abstracted type specified by GoSpeech2Text.
	//
	// If the array is left empty (array with zero values), nothing will be redacted. // TODO fact check; default value.
	// Currently, the array can have a maximum of 11 entries, according to AWS docs (https://docs.aws.amazon.com/transcribe/latest/APIReference/API_ContentRedaction.html).
	// However, this limit is not enforced by GoSpeech2Text, because this limit might change in the future.
	// If AWS adds new possible redaction entities, you can specify them using a string instead of the
	// RedactionEntityType enum values defined by GoSpeech2Text.
	//
	// Use RedactionEntityAll to redact all possible kinds of personally identifiable information.
	RedactionEntityTypes []*RedactionEntityType
	// RedactionOutput specifies if only the redacted transcript, or both the redacted and unredacted transcripts are returned.
	// TODO check if possible; how are files stored?
	RedactionOutput RedactionOutput
}

type ContentRedactionType string

const (
	RedactionTypePersonallyIdentifiableInformation ContentRedactionType = "PII"
)

type RedactionEntityType string

const (
	RedactionEntityBankAccountNumber RedactionEntityType = "BANK_ACCOUNT_NUMBER"
	RedactionEntityBankRouting       RedactionEntityType = "BANK_ROUTING"
	RedactionEntityCreditDebitNumber RedactionEntityType = "CREDIT_DEBIT_NUMBER"
	RedactionEntityCreditDebitCvv    RedactionEntityType = "CREDIT_DEBIT_CVV"
	RedactionEntityCreditDebitExpiry RedactionEntityType = "CREDIT_DEBIT_EXPIRY"
	RedactionEntityPin               RedactionEntityType = "PIN"
	RedactionEntityEmail             RedactionEntityType = "EMAIL"
	RedactionEntityAddress           RedactionEntityType = "ADDRESS"
	RedactionEntityName              RedactionEntityType = "NAME"
	RedactionEntityPhone             RedactionEntityType = "PHONE"
	RedactionEntitySsn               RedactionEntityType = "SSN"
	RedactionEntityAll               RedactionEntityType = "ALL"
)

type RedactionOutput string

const (
	RedactionOutputRedacted              RedactionOutput = "redacted"
	RedactionOutputRedactedAndUnredacted RedactionOutput = "redacted_and_unredacted"
)

// TranscriptionJobNameConfig is only used on AWS
type TranscriptionJobNameConfig struct {
	// TranscriptionJobName is the name of the transcription job that is created.
	// From AWS docs: "This name is case-sensitive, cannot contain spaces, and must be unique within an
	// Amazon Web Services account."
	// Every transcription job on AWS needs a unique name. If TranscriptionJobName is left empty (""),
	// just the current timestamp (nanoseconds) will be used as job name
	// (regardless if AppendCurrentTimestamp is true or false).
	// Other than that, the TranscriptionJobName is not checked for errors.
	TranscriptionJobName string
	// AppendCurrentTimestamp says if the current timestamp, at which the transcription job is created, should be
	// appended to the TranscriptionJobName. Appending the current timestamp makes it easier to create unique
	// transcription job names. It also allows developers to re-use their options object for multiple transcription jobs,
	// without having to specify a new transcription job name every time.
	// The timestamp is the number of nanoseconds since January 1, 1970, UTC
	AppendCurrentTimestamp bool
}

func (jobNameConfig TranscriptionJobNameConfig) GetTranscriptionJobName() string {
	if (jobNameConfig.TranscriptionJobName != "") && !jobNameConfig.AppendCurrentTimestamp {
		return jobNameConfig.TranscriptionJobName
	} else {
		now := time.Now()
		return jobNameConfig.TranscriptionJobName + strconv.FormatInt(now.UnixNano(), 10)
	}
}
