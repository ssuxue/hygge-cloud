package service

import (
	"errors"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"io/ioutil"
)

var endpoint string = "http://oss-cn-beijing.aliyuncs.com"
var accessKeyId string = "LTAI4G2Z3WBiKdUYs8hKMYi5"
var accessKeySecret string = "pW10GhV4YfWN0yrxtqvu8NUHksAzGB"
var bucketName string = "chase-oss1"

type OSSClient struct {
	ossClient *oss.Client
}

// Return OSS client.
func NewOSSClient() (*OSSClient, error) {
	client, err := oss.New(endpoint, accessKeyId, accessKeySecret)
	if err != nil {
		return nil, err
	}

	// Judge the bucket whether exists.
	isExist, err := client.IsBucketExist("<yourBucketName>")
	if err != nil {
		return nil, err
	}

	if !isExist {
		// If the bucket doesn't exist, create bucket.
		err = client.CreateBucket(bucketName)
		if err != nil {
			return nil, err
		}
	}
	return &OSSClient{client}, nil
}

// Upload file to OSS.
func (cli *OSSClient) Upload(filePath string) error {
	bucket, err := cli.ossClient.Bucket(bucketName)
	if err != nil {
		return errors.New("there is not bucket named: " + bucketName)
	}

	err = bucket.PutObjectFromFile(bucketName, filePath)
	if err != nil {
		return err
	}
	return nil
}

// Download file to local.
func (cli *OSSClient) Download(path string) error {
	bucket, err := cli.ossClient.Bucket(bucketName)
	if err != nil {
		return errors.New("there is not bucket named: " + bucketName)
	}

	err = bucket.GetObjectToFile(bucketName, path)
	if err != nil {
		return err
	}
	return nil
}

func (cli *OSSClient) SteamDownload() (string, error) {
	bucket, err := cli.ossClient.Bucket(bucketName)
	if err != nil {
		return "", errors.New("there is not bucket named: " + bucketName)
	}

	body, err := bucket.GetObject("<yourObjectName>")
	if err != nil {
		return "", err
	}

	defer body.Close()

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
