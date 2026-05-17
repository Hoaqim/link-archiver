package queue

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type sqsAPI interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
	GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
}

const longPollWaitSeconds int32 = 20

type SQSQueue struct {
	client   sqsAPI
	queueURL string
}

type SQSMessage struct {
	queue         *SQSQueue
	body          []byte
	receiptHandle string
	receiveCount  int
}

var (
	_ Queue   = (*SQSQueue)(nil)
	_ Message = (*SQSMessage)(nil)
)

func NewSQS(ctx context.Context, queueURL string) (*SQSQueue, error) {
	if queueURL == "" {
		return nil, fmt.Errorf("queue URL empty")
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config load: %w", err)
	}

	q := &SQSQueue{
		client:   sqs.NewFromConfig(cfg),
		queueURL: queueURL,
	}

	if err := q.Ping(ctx); err != nil {
		return nil, fmt.Errorf("sqs ping queue %w", err)
	}
	return q, nil
}

func (q *SQSQueue) Enqueue(ctx context.Context, payload []byte) error {
	_, err := q.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(q.queueURL),
		MessageBody: aws.String(string(payload)),
	})

	if err != nil {
		return fmt.Errorf("sqs send message: %w", err)
	}
	return nil
}

func (q *SQSQueue) Dequeue(ctx context.Context) (Message, error) {
	out, err := q.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(q.queueURL),
		MaxNumberOfMessages: 1,
		WaitTimeSeconds:     longPollWaitSeconds,
		MessageSystemAttributeNames: []sqstypes.MessageSystemAttributeName{
			sqstypes.MessageSystemAttributeNameApproximateReceiveCount,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("sqs receive message: %w", err)
	}

	if len(out.Messages) == 0 {
		return nil, ErrNoMessage
	}

	m := out.Messages[0]
	return &SQSMessage{
		queue:         q,
		body:          []byte(aws.ToString(m.Body)),
		receiptHandle: aws.ToString(m.ReceiptHandle),
		receiveCount:  parseReceiveCount(m.Attributes),
	}, nil
}

func (q *SQSQueue) Ping(ctx context.Context) error {
	_, err := q.client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: aws.String(q.queueURL),
		AttributeNames: []sqstypes.QueueAttributeName{
			sqstypes.QueueAttributeNameQueueArn,
		},
	})

	if err != nil {
		return fmt.Errorf("sqs ping: %w", err)
	}

	return nil
}

func (q *SQSQueue) Close() error { return nil }

func (m *SQSMessage) Payload() []byte   { return m.body }
func (m *SQSMessage) ReceiveCount() int { return m.receiveCount }

func (m *SQSMessage) Ack(ctx context.Context) error {
	_, err := m.queue.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(m.queue.queueURL),
		ReceiptHandle: aws.String(m.receiptHandle),
	})
	if err != nil {
		return fmt.Errorf("sqs message ack: %w", err)
	}
	return nil
}

func (m *SQSMessage) Nack(ctx context.Context) error { return nil }

func parseReceiveCount(attrs map[string]string) int {
	v, ok := attrs[string(sqstypes.MessageSystemAttributeNameApproximateReceiveCount)]
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}
