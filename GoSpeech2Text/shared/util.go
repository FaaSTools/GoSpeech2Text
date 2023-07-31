package shared

import (
	"errors"
	"fmt"
	"github.com/FaaSTools/GoStorage/gostorage"
	"io"
	"os"
	"strings"
)

func IsAWSUrl(urlString string) bool {
	return strings.HasPrefix(urlString, "s3://") || (strings.HasPrefix(urlString, "https://") && strings.Contains(urlString, "s3"))
}

// IsGoogleUrl Google Object URL: gs://gostorage-bucket-test/test.png
// Google Object URL: https://storage.cloud.google.com/gostorage-bucket-test/test.png
// Taken from GoStorage
func IsGoogleUrl(urlString string) bool {
	return strings.HasPrefix(urlString, "gs://") || strings.Contains(urlString, "storage.cloud.google.com")
}

const DefaultAWSRegion = "us-east-1"

// ParseAWSUrl AWS Object URL (with explicit region)
// Taken from GoStorage
func ParseAWSUrl(urlString string) gostorage.GoStorageObject {
	var bucket string
	var key string
	var region string

	urlString = urlString[strings.Index(urlString, "https://")+len("https://"):]
	bucket = urlString[:strings.Index(urlString, ".")]
	urlString = urlString[strings.Index(urlString, ".")+len(".s3."):]
	if strings.HasPrefix(urlString, "amazonaws.com") { //No region specified
		region = DefaultAWSRegion
	} else {
		region = urlString[:strings.Index(urlString, ".")]
		urlString = urlString[strings.Index(urlString, ".")+1:]
	}
	urlString = urlString[strings.Index(urlString, "amazonaws.com")+len("amazonaws.com"):]
	if strings.HasPrefix(urlString, "/") {
		urlString = urlString[1:]
	}
	key = urlString
	return gostorage.GoStorageObject{Bucket: bucket, Key: key, Region: region, ProviderType: gostorage.ProviderAWS}
}

// ParseGoogleUrl Google Object URL
// Taken from GoStorage
func ParseGoogleUrl(urlString string) gostorage.GoStorageObject {
	var bucket string
	var key string

	if strings.HasPrefix(urlString, "gs://") {
		urlString = urlString[strings.Index(urlString, "gs://")+len("gs://"):]
	} else if strings.HasPrefix(urlString, "https://storage.cloud.google.com/") {
		urlString = urlString[strings.Index(urlString, "https://storage.cloud.google.com/")+len("https://storage.cloud.google.com/"):]
	}
	if strings.Contains(urlString, "/") {
		bucket = urlString[:strings.Index(urlString, "/")]
		key = urlString[strings.Index(urlString, "/")+1:]
	} else {
		bucket = urlString
	}
	return gostorage.GoStorageObject{Bucket: bucket, Key: key, ProviderType: gostorage.ProviderGoogle}
}

// StringToReader Taken from https://code-maven.com/slides/golang/create-io-reader-from-string
func StringToReader(str string) (io.Reader, error) {
	myReader := strings.NewReader(str)
	buffer := make([]byte, 1024)
	for {
		_, err := myReader.Read(buffer)
		if err != nil {
			if err != io.EOF {
				return myReader, err
			}
			break
		}
	}
	return myReader, nil
}

// from GoStorage
func checkErr(err interface{}, msg interface{}) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", msg)
		os.Exit(1)
	}
}

// ParseUrlToGoStorageObject parses Object/Bucket URLs from AWS and Google to extract information such as bucketName, key, region etc.
// Taken from GoStorage
func ParseUrlToGoStorageObject(urlString string) gostorage.GoStorageObject {
	if IsAWSUrl(urlString) {
		return ParseAWSUrl(urlString)
	} else if IsGoogleUrl(urlString) {
		return ParseGoogleUrl(urlString)
	} else {
		if _, err := os.Stat(urlString); errors.Is(err, os.ErrNotExist) {
			checkErr(err, fmt.Sprintf("unable to find local file from {%v}, Error: %v", urlString, err))
		}
		return gostorage.GoStorageObject{IsLocal: true, LocalFilePath: urlString}
	}
}

// GetFileTypeFromFileName returns the file type (i.e. file extension) if the given fileName.
// fileName can also be a path or URL.
// If there are multiple file extensions (example: 'test_file.tar.gz'), only the last file extension is returned ('gz').
func GetFileTypeFromFileName(fileName string) string {
	splits := strings.SplitAfter(fileName, ".")
	if len(splits) < 2 { // if splits is < 2, it means no file type; if splits is < 1, it means that fileName was empty
		return ""
	}
	return splits[len(splits)-1]
}

func AudioToFile(reader io.ReadCloser) (string, error) {
	// TODO
	return "", nil
}
