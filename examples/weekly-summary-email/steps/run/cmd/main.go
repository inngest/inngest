package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/actionsdk"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

const (
	FromEmail = "noreply@example.com"
	FromName  = "Your Service"
	Subject   = "Your [service] weekly summary"
	// TemplateID is the sendgrid template that we're sending
	TemplateID = "d-c6dcf1f72bdd4beeb15a9aa6c72fcd2c"
)

func main() {
	n, err := Run(context.Background())
	if err != nil {
		actionsdk.WriteResult(&actionsdk.Result{
			Body: map[string]any{
				"sent":  n,
				"error": err.Error(),
			},
			Status: 500, // A status of 5xx retries the function.
		})
		os.Exit(1)
	}

	actionsdk.WriteResult(&actionsdk.Result{
		Body: map[string]any{
			"sent": n,
		},
		Status: 200,
	})
}

// Run sends a weekly summary email for each account fetched from the
// SummaryFetcher.
//
// You can access internal APIs, databases, and other data sources as
// you'd expect from normal functions to fetch data in the real world.
//
// This returns the number of summaries sent.
func Run(ctx context.Context) (int, error) {
	all, err := NewSummaryFetcher().Fetch(ctx)
	if err != nil {
		return 0, err
	}

	for n, s := range all {
		// Send this summary.
		//
		// An alternative strategy here is to send another event
		// that triggers a new function to send this summary.  This
		// ensures that retries are local to the specific summary
		// email being sent.
		if err := Send(ctx, s); err != nil {
			return n, err
		}

		// Send a "summary/user.fetched" event to Inngest.  We can
		// then create a new function that is triggered by this event
		// to send the email vs doing it here.
		//
		// This gives logs per-user, allows retries per-user, and allows
		// us to handle SendAt natively via waits within our function:
		// https://www.inngest.com/docs/functions/step-functions#after-configuration
		inngestgo.NewClient(os.Getenv("INNGEST_EVENT_KEY")).Send(ctx, inngestgo.Event{
			Name: "summary/user.fetched",
			Data: map[string]any{
				"name":    s.Name,
				"email":   s.Email,
				"usage":   s.Usage,
				"notes":   s.Notes,
				"send_at": s.SendAt,
			},
			User: map[string]any{
				"email": s.Email,
				"name":  s.Name,
			},
		})
	}

	return len(all), nil
}

type SummaryFetcher interface {
	Fetch(ctx context.Context) ([]Summary, error)
}

type Summary struct {
	Name  string
	Email string
	Usage int
	Notes string
	// SendAt allows us to record local preferences for timezones.
	SendAt time.Time
}

// NewSummaryFetcher returns a SummaryFetcher used to send all summaries.
func NewSummaryFetcher() SummaryFetcher {
	// XXX: Replace this with your own implementation.
	return mockfetcher{}
}

func Send(ctx context.Context, s Summary) error {
	from := mail.NewEmail(FromName, FromEmail)
	to := mail.NewEmail(s.Name, s.Email)

	// Create a personalization, which fills template data for this summary.
	p := mail.NewPersonalization()
	p.AddTos(to)
	p.SetDynamicTemplateData("usage", s.Usage)
	p.SetDynamicTemplateData("notes", s.Notes)

	// Create and send the email.
	e := mail.NewV3MailInit(from, Subject, to)
	e.SetTemplateID(TemplateID)
	e.AddPersonalizations(p)
	e.SetSendAt(int(s.SendAt.Unix()))

	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	if _, err := client.Send(e); err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	return nil
}
