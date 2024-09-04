package emailer

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"golang.org/x/exp/slices"
)

var (
	ErrDial          = errors.New("failed to dial IMAP server")
	ErrLogin         = errors.New("failed to login to IMAP server")
	ErrMailbox       = errors.New("failed to select INBOX mailbox")
	ErrSearch        = errors.New("failed to search for unread messages")
	ErrFetchBody     = errors.New("FETCH command did not return body section")
	ErrReader        = errors.New("failed to create body reader")
	ErrNoBody        = errors.New("failed to parse a body")
	ErrPart          = errors.New("failed to get next part")
	ErrReadBody      = errors.New("failed to read body")
	ErrReadHeader    = errors.New("failed to read header value")
	ErrInvalidSender = errors.New("invalid sender address")
)

type Querier interface {
	Fetch() ([]*Entry, error)
}

type Entry struct {
	Subject   string    `json:"subject"`
	Body      []byte    `json:"body"`
	CreatedOn time.Time `json:"created_on"`
}

// BodyParser can be used to do postprocessing of the body part, for example parsing the html body.
type BodyParser interface {
	Parse(body []byte) ([]byte, error)
}

type Client struct {
	client         *imapclient.Client
	username       string
	password       string
	allowedSenders []string
	bodyParser     BodyParser
}

func NewClient(host string, username string, password string, allowedSenders []string, parser BodyParser) (*Client, error) {
	client, errDial := imapclient.DialTLS(host, nil)
	if errDial != nil {
		return nil, errors.Join(errDial, ErrDial)
	}

	if errLogin := client.Login(username, password).Wait(); errLogin != nil {
		return nil, errors.Join(errLogin, ErrLogin)
	}

	return &Client{
		client:         client,
		username:       username,
		password:       password,
		allowedSenders: stringutil.ToLowerSlice(allowedSenders),
		bodyParser:     parser,
	}, nil
}

func (o Client) Close() {
	if o.client != nil {
		if err := o.client.Logout().Wait(); err != nil {
			slog.Debug("Failed to logout of imap client", log.ErrAttr(err))
		}
		if err := o.client.Close(); err != nil {
			slog.Error("Failed to close imap connection", log.ErrAttr(err))
		}
	}
}

func (o Client) Fetch() ([]*Entry, error) {
	_, errSelect := o.client.Select("INBOX", nil).Wait()
	if errSelect != nil {
		return nil, errors.Join(errSelect, ErrMailbox)
	}

	searchData, errSearch := o.client.UIDSearch(&imap.SearchCriteria{
		NotFlag: []imap.Flag{imap.FlagSeen},
	}, nil).Wait()
	if errSearch != nil {
		return nil, errors.Join(errSearch, ErrSearch)
	}

	fetchCmd := o.client.Fetch(searchData.All, &imap.FetchOptions{Envelope: true, BodySection: []*imap.FetchItemBodySection{
		{},
	}})

	defer func(fetchCmd *imapclient.FetchCommand) {
		err := fetchCmd.Close()
		if err != nil {
			slog.Error("Failed to close imap fetch command", log.ErrAttr(err))
		}
	}(fetchCmd)

	var entries []*Entry

	for {
		message := fetchCmd.Next()
		if message == nil {
			break
		}

		entry, errBody := o.readMessage(message)
		if errBody != nil {
			if errors.Is(errBody, ErrInvalidSender) {
				continue
			}

			return nil, errBody
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (o Client) isAllowedSender(mailReader *mail.Reader) error {
	if len(o.allowedSenders) > 0 {
		addresses, errAddresses := mailReader.Header.AddressList("From")
		if errAddresses != nil {
			return fmt.Errorf("%w: Addresses", ErrReadHeader)
		}

		validAddr := false
		for _, addr := range addresses {
			if slices.Contains(o.allowedSenders, strings.ToLower(addr.Address)) {
				validAddr = true

				break
			}
		}

		if !validAddr {
			return ErrInvalidSender
		}
	}

	return nil
}

func (o Client) messageReader(message *imapclient.FetchMessageData) (*mail.Reader, error) {
	// Find the body section in the response
	var (
		bodySection imapclient.FetchItemDataBodySection
		valid       = false
	)

	for {
		item := message.Next()
		if item == nil {
			break
		}
		bodySection, valid = item.(imapclient.FetchItemDataBodySection)
		if valid {
			break
		}
	}

	if !valid {
		return nil, ErrFetchBody
	}

	mailReader, err := mail.CreateReader(bodySection.Literal)
	if err != nil {
		return nil, errors.Join(err, ErrReader)
	}

	return mailReader, nil
}

func (o Client) readMessage(message *imapclient.FetchMessageData) (*Entry, error) {
	mailReader, errReader := o.messageReader(message)
	if errReader != nil {
		return nil, errReader
	}

	if errSender := o.isAllowedSender(mailReader); errSender != nil {
		return nil, errSender
	}

	subject, errSubject := mailReader.Header.Subject()
	if errSubject != nil {
		return nil, fmt.Errorf("%w: Subject", ErrReadHeader)
	}

	createdOn, errCreatedOn := mailReader.Header.Date()
	if errCreatedOn != nil {
		return nil, fmt.Errorf("%w: Date", ErrReadHeader)
	}

	// Process the message's parts
	for {
		nextPart, errPart := mailReader.NextPart()
		if errors.Is(errPart, io.EOF) {
			break
		} else if errPart != nil {
			return nil, errors.Join(errPart, ErrPart)
		}

		if _, ok := nextPart.Header.(*mail.InlineHeader); ok {
			body, errRead := io.ReadAll(nextPart.Body)
			if errRead != nil {
				return nil, errors.Join(errRead, ErrReadBody)
			}

			if o.bodyParser != nil {
				parsedBody, errParse := o.bodyParser.Parse(body)
				if errParse != nil {
					return nil, errParse
				}
				body = parsedBody
			}

			return &Entry{Subject: subject, Body: body, CreatedOn: createdOn}, nil
		}
	}

	return nil, ErrNoBody
}
