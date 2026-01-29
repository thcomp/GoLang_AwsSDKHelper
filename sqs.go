package awssdkhelper

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/rs/xid"

	ThcompUtility "github.com/thcomp/GoLang_Utility"
)

type SQSHelper struct {
	client                     *sqs.Client
	queueURL                   string
	fifo                       bool
	messageGroupID             *string
	nextMessageDeduplicationId *string
}

func NewSQSHelperWithRole(queueURL, region string) (ret *SQSHelper) {
	if sdkConfig, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
	); err == nil {
		ret = &SQSHelper{
			queueURL: queueURL,
			client:   sqs.NewFromConfig(sdkConfig),
			fifo:     strings.HasSuffix(queueURL, ".fifo"),
		}
	}

	return ret
}

func NewSQSHelperWithIAM(queueURL, region, accessKeyId, secretAccessKey string) (ret *SQSHelper) {
	if sdkConfig, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				accessKeyId,
				secretAccessKey,
				``,
			),
		),
	); err == nil {
		ret = &SQSHelper{
			queueURL: queueURL,
			client:   sqs.NewFromConfig(sdkConfig),
			fifo:     strings.HasSuffix(queueURL, ".fifo"),
		}
	}

	return ret
}

func (helper *SQSHelper) SetMessageGroupID(messageGroupID string) (ret bool) {
	if helper.fifo {
		helper.messageGroupID = &messageGroupID
		ret = true
	}

	return
}

func (helper *SQSHelper) SetNextMessageDeduplicationID(nextMessageDeduplicationID string) (ret bool) {
	if helper.fifo {
		helper.nextMessageDeduplicationId = &nextMessageDeduplicationID
		ret = true
	}

	return
}

func (helper *SQSHelper) SendMessage(message string) (id, seqNum string, ret error) {
	if helper.client != nil {
		nextMessageDeduplicationId := helper.nextMessageDeduplicationId

		if helper.fifo {
			helper.nextMessageDeduplicationId = nil

			if helper.messageGroupID == nil {
				helper.messageGroupID = ThcompUtility.ToStringPointer(xid.New().String())
			}
			if nextMessageDeduplicationId == nil {
				nextMessageDeduplicationId = ThcompUtility.ToStringPointer(xid.New().String())
			}
		}

		ThcompUtility.LogfV("send Message: Message Group ID: %v, Message Deduplication ID: %v", helper.messageGroupID, nextMessageDeduplicationId)
		if output, err := helper.client.SendMessage(
			context.Background(),
			&sqs.SendMessageInput{
				MessageBody:            aws.String(message),
				QueueUrl:               &helper.queueURL,
				MessageGroupId:         helper.messageGroupID,
				MessageDeduplicationId: nextMessageDeduplicationId,
			},
		); err == nil {
			id = *output.MessageId

			if output.SequenceNumber != nil {
				seqNum = *(output.SequenceNumber)
			}
		} else {
			ret = err
		}
	} else {
		ret = fmt.Errorf("not created by NewSQSHelper")
	}

	return
}
