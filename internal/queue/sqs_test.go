package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type mockSQS struct {
	sendCalled   int
	sendInput    *sqs.SendMessageInput
	receiveOut   *sqs.ReceiveMessageOutput
	receiveErr   error
	deleteCalled int
	deleteInput  *sqs.DeleteMessageInput
}

func (m *mockSQS) SendMessage(ctx context.Context, in *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	m.sendCalled++
	m.sendInput = in
	return &sqs.SendMessageOutput{}, nil
}
func (m *mockSQS) ReceiveMessage(ctx context.Context, _ *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if m.receiveErr != nil {
		return nil, m.receiveErr
	}
	if m.receiveOut != nil {
		return m.receiveOut, nil
	}
	return &sqs.ReceiveMessageOutput{}, nil
}
func (m *mockSQS) DeleteMessage(ctx context.Context, in *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	m.deleteCalled++
	m.deleteInput = in
	return &sqs.DeleteMessageOutput{}, nil
}
func (m *mockSQS) GetQueueAttributes(ctx context.Context, _ *sqs.GetQueueAttributesInput, _ ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	return &sqs.GetQueueAttributesOutput{}, nil
}

const testQueueURL = "https://sqs.example.com/123/test-queue"

func newTestQueue(client sqsAPI) *SQSQueue {
	return &SQSQueue{client: client, queueURL: testQueueURL}
}

func TestSQS_EnqueueSendsPayload(t *testing.T) {
	mock := &mockSQS{}
	q := newTestQueue(mock)

	if err := q.Enqueue(context.Background(), []byte(`{"id":"abc"}`)); err != nil {
		t.Fatal(err)
	}
	if mock.sendCalled != 1 {
		t.Errorf("sendCalled = %d, want 1", mock.sendCalled)
	}
	if got := aws.ToString(mock.sendInput.MessageBody); got != `{"id":"abc"}` {
		t.Errorf("MessageBody = %q, want the payload", got)
	}
}

func TestSQS_DequeueEmptyReturnsErrNoMessage(t *testing.T) {
	q := newTestQueue(&mockSQS{})

	_, err := q.Dequeue(context.Background())
	if !errors.Is(err, ErrNoMessage) {
		t.Errorf("err = %v, want ErrNoMessage", err)
	}
}

func TestSQS_DequeueReturnsMessageWithReceiveCount(t *testing.T) {
	mock := &mockSQS{
		receiveOut: &sqs.ReceiveMessageOutput{
			Messages: []sqstypes.Message{{
				Body:          aws.String(`{"id":"abc","url":"x"}`),
				ReceiptHandle: aws.String("receipt-123"),
				Attributes: map[string]string{
					string(sqstypes.MessageSystemAttributeNameApproximateReceiveCount): "3",
				},
			}},
		},
	}
	q := newTestQueue(mock)

	msg, err := q.Dequeue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := string(msg.Payload()); got != `{"id":"abc","url":"x"}` {
		t.Errorf("payload = %q", got)
	}
	if msg.ReceiveCount() != 3 {
		t.Errorf("ReceiveCount = %d, want 3", msg.ReceiveCount())
	}
}

func TestSQS_AckCallsDeleteMessage(t *testing.T) {
	mock := &mockSQS{}
	q := newTestQueue(mock)
	msg := &SQSMessage{queue: q, receiptHandle: "receipt-xyz"}

	if err := msg.Ack(context.Background()); err != nil {
		t.Fatal(err)
	}
	if mock.deleteCalled != 1 {
		t.Errorf("deleteCalled = %d, want 1", mock.deleteCalled)
	}
	if got := aws.ToString(mock.deleteInput.ReceiptHandle); got != "receipt-xyz" {
		t.Errorf("ReceiptHandle = %q, want receipt-xyz", got)
	}
}

func TestSQS_NackDoesNotDelete(t *testing.T) {
	mock := &mockSQS{}
	q := newTestQueue(mock)
	msg := &SQSMessage{queue: q, receiptHandle: "receipt-xyz"}

	if err := msg.Nack(context.Background()); err != nil {
		t.Fatal(err)
	}
	if mock.deleteCalled != 0 {
		t.Errorf("Nack called DeleteMessage %d times, want 0 — Nack must be a no-op for SQS's redrive policy to work",
			mock.deleteCalled)
	}
}
