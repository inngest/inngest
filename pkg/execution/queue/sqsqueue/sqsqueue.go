package sqsqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/config/registration"
	"github.com/inngest/inngest-cli/pkg/execution/queue"
	"github.com/inngest/inngest-cli/pkg/logger"
	"github.com/inngest/inngest-cli/pkg/pubsub"
)

func init() {
	registration.RegisterQueue(&Config{})
}

type Config struct {
	Region   string
	QueueURL string
	Topic    string
	// Concurrency represents the number of items to process concurrently.
	Concurrency int
}

func (Config) QueueName() string { return "aws-sqs" }

func (c *Config) Queue() (queue.Queue, error) {
	url, err := url.Parse(c.QueueURL)
	if err != nil {
		return nil, err
	}

	i := &impl{
		config:   *c,
		urlQuery: url.Query(),
	}

	// Remove this, so that we don't have query strings in our url
	// when manually sending.
	url.RawQuery = ""
	i.url = url.String()

	err = i.connect(context.Background())
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

	// qURL is the parsed and formatted queue URL with no
	// query strings
	url      string
	urlQuery url.Values

	sess *session.Session
	sqs  *sqs.SQS
}

func (i *impl) connect(ctx context.Context) error {
	var err error

	sessionCfg := &aws.Config{
		Region: aws.String(i.config.Region),
	}

	// Handle sessions, including endpoints, using a similar
	// strategy to gocloud.
	for param, values := range i.urlQuery {
		value := values[0]
		switch param {
		case "region":
			if i.config.Region != "" && value != i.config.Region {
				return fmt.Errorf("Conflicting regions in config and queue URL")
			}
			sessionCfg.Region = aws.String(value)
		case "endpoint":
			sessionCfg.Endpoint = aws.String(value)
		}
	}

	i.sess, err = session.NewSession(sessionCfg)
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
	byt, err := json.Marshal(w)
	if err != nil {
		return fmt.Errorf("error marshalling sqs queue wrapper: %w", err)
	}

	// Wrap this in a pubsub.Message, allowing us to use the message fields
	// as a wrapper with our own visibility timeout - plus our own Message
	// wrapper.
	msg := pubsub.Message{
		Name:      "queue/item",
		Data:      string(byt),
		Timestamp: at,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error marshalling sqs queue message: %w", err)
	}

	logger.From(ctx).
		Trace().
		Interface("delay", w.DelaySeconds()).
		Interface("payload", msg).
		Msg("enqueued step via sqs")

	_, err = i.sqs.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: w.DelaySeconds(),
		MessageBody:  aws.String(string(payload)),
		QueueUrl:     aws.String(i.url),
	})

	return err
}

// Run subscribes to the topic, processing each queue item.
func (i impl) Run(ctx context.Context, f func(context.Context, queue.Item) error) error {
	// We can use our pubsub broker logic here, as SQS is a supported backend.
	sqsConf := config.SQSMessaging{
		Region:   i.config.Region,
		Topic:    i.config.Topic,
		QueueURL: i.config.QueueURL,
	}
	conf := config.MessagingService{
		Backend:  sqsConf.Backend(),
		Concrete: sqsConf,
	}
	sub, err := pubsub.NewSubscriber(ctx, conf)
	if err != nil {
		return fmt.Errorf("error opening sqs queue: %w", err)
	}
	return sub.SubscribeN(ctx, i.config.Topic, func(ctx context.Context, m pubsub.Message) error {
		if m.Name != "queue/item" {
			return fmt.Errorf("unknown queue event type: %s", m.Name)
		}

		w := &Wrapper{}
		if err := json.Unmarshal([]byte(m.Data), w); err != nil {
			return fmt.Errorf("error unmarshalling queue item: %w", err)
		}

		logger.From(ctx).
			Trace().
			Interface("delay", w.DelaySeconds()).
			Interface("payload", w).
			Msg("received step via sqs")

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
