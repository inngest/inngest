package sqsqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/config/registration"
	"github.com/inngest/inngest-cli/pkg/execution/queue"
	"github.com/inngest/inngest-cli/pkg/pubsub"
)

func init() {
	registration.RegisterQueue(&Config{})
}

type Config struct {
	Region    string
	AccountID string
	Topic     string
	// Concurrency represents the number of items to process concurrently.
	Concurrency int
}

func (c Config) QueueURL() string {
	return fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s", c.Region, c.AccountID, c.Topic)
}

func (Config) QueueName() string { return "sqs" }

func (c *Config) Queue() (queue.Queue, error) {
	i := &impl{
		config: *c,
	}

	err := i.connect(context.Background())
	return i, err
}

func (c *Config) Producer() (queue.Producer, error) {
	return c.Queue()
}

func (c *Config) Consumer() (queue.Consumer, error) {
	return c.Queue()
}

type impl struct {
	config Config

	sess *session.Session
	sqs  *sqs.SQS
}

func (i *impl) connect(ctx context.Context) error {
	var err error
	i.sess, err = session.NewSession(&aws.Config{
		Region: aws.String(i.config.Region),
	})
	if err != nil {
		return err
	}
	i.sqs = sqs.New(i.sess)
	return nil
}

func (i impl) Enqueue(ctx context.Context, item queue.Item, at time.Time) error {
	// Wrap the item with our own custom message.
	w := Wrapper{
		At:   at,
		Item: item,
	}
	byt, err := json.Marshal(item)
	if err != nil {
		return nil
	}

	// Wrap this in a pubsub.Message, allowing us to use the message fields
	// as a wrapper with our own visibility timeout - plus our own Message
	// wrapper.
	msg := pubsub.Message{
		Name:      "queue/item",
		Version:   "1",
		Data:      string(byt),
		Timestamp: at,
	}
	byt, err = json.Marshal(msg)
	if err != nil {
		return nil
	}

	_, err = i.sqs.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: w.DelaySeconds(),
		MessageBody:  aws.String(string(byt)),
		QueueUrl:     aws.String(i.config.QueueURL()),
	})

	return err
}

// Run subscribes to the topic, processing each queue item.
func (i impl) Run(ctx context.Context, f func(context.Context, queue.Item) error) error {
	// We can use our pubsub broker logic here, as SQS is a supported backend.
	sqsConf := config.SQSMessaging{
		Region:    i.config.Region,
		AccountID: i.config.AccountID,
		Topic:     i.config.Topic,
	}
	conf := config.MessagingService{
		Backend:  sqsConf.Backend(),
		Concrete: sqsConf,
	}
	sub, err := pubsub.NewSubscriber(ctx, conf)
	if err != nil {
		return err
	}
	return sub.SubscribeN(ctx, i.config.Topic, func(ctx context.Context, m pubsub.Message) error {
		// TODO: Handle the incoming message here.
		if m.Name != "queue/item" {
			return fmt.Errorf("unknown queue event type: %s", m.Name)
		}

		w := &Wrapper{}
		if err := json.Unmarshal([]byte(m.Data), w); err != nil {
			return fmt.Errorf("error unmarshalling queue item: %w", err)
		}

		if w.At.After(time.Now()) {
			// Re-enqueue this at a future time.
			return i.Enqueue(ctx, w.Item, w.At)
		}

		return f(ctx, w.Item)
	}, int64(i.config.Concurrency))
}

// Wrapper represents a single message sent across SQS, wrapping a queue.Item
// to provide SQS-specific functionality.
//
// For example, SQS only allows messages with up to a 15 minute delay, whereas
// we can enqueue messages at any point in the future.  This wrapper allows us
// to sepcify our own visibility timeouts which are handled independently
// from SQS.
type Wrapper struct {
	// V represents the message version, allowing us to change this struct
	// definition over time.
	V int `json:"v"`
	// At represents the implicit visibility for this message.  If this is
	// after Now(), the message must be re-enqueued for min(15min, delta)
	// to support enqueuing messages at some point in the future.
	At time.Time `json:"at"`
	// Item represents the actual queue item
	Item queue.Item `json:"item"`
}

func (m Wrapper) DelaySeconds() *int64 {
	diff := time.Until(m.At)
	if diff <= 0 {
		return nil
	}
	if diff > 15*time.Minute {
		return aws.Int64(15 * 60)
	}
	secs := int64(diff.Seconds())
	return &secs
}
