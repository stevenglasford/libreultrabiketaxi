///This is for ease of use when trying to submit an email message to the ZOLEO device.
// message := []byte("From: " + smtpUsername + "\r\n" +
// "To: " + receiverEmail + "\r\n" +
// "Subject: initContext in libretaxi.go\r\n" +
// "\r\n" +
// "This is a test email sent from a Go program.\r\n")

// auth := smtp.PlainAuth("", smtpUsername, smtpToken, smtpServer)

// // Sending email.

// if err := smtp.SendMail(smtpServer+":"+ strconv.FormatInt(smtpPort, 10), auth, smtpUsername, []string{receiverEmail}, message); err != nil {
// fmt.Println(err)
// return nil
// }
// fmt.Println("Email sent successfully")

/// When retrieving the email via imap ensure that the email has been signed and encrypted.
///Emails sent to the zoleo devices will be encrypted to ensure that no one is messing with the server and riders


package main

import (
	"database/sql"
	// "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/leonelquinteros/gotext"
	_ "github.com/lib/pq" // important
	// "go.uber.org/ratelimit"
	// "libretaxi/callback"
	"libretaxi/config"
	"libretaxi/context"
	// "libretaxi/menu"
	"libretaxi/rabbit"
	"libretaxi/repository"
	"libretaxi/sender"
	"log"
	"math/rand"
	"time"
	"net/smtp"
	"fmt"
	"strconv"
	"errors"
	"regexp"
	"strings"
	"github.com/emersion/go-imap"
    "github.com/emersion/go-imap/client"
	"unicode"
	"io/ioutil"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"bytes"
	"encoding/json"
)

type RoutingRequest struct {
	Client             string    `json:"client"`
	Origin             Location  `json:"origin"`
	Destination        Location  `json:"destination"`
	Settings           Settings  `json:"settings"`
	DepartureDateTime  string    `json:"departureDateTime"`
	Key                string    `json:"key"`
	Waypoints          []Location `json:"waypoints"`
	UID                *string   `json:"uid"` // pointer to handle null value
}

type Location struct {
	// LocationType string   `json:"locationType"`
	Point        Point    `json:"point"`
}

type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Settings struct {
	BikeType                string   `json:"bikeType"`
	AverageSpeed            int      `json:"averageSpeed"`
	AllowedTransportModes   []string `json:"allowedTransportModes"`
	Stairs                  string   `json:"stairs"`
	Pavements               string   `json:"pavements"`
	Oneways                 string   `json:"oneways"`
	Traffic                 string   `json:"traffic"`
	Surface                 string   `json:"surface"`
	Climbs                  string   `json:"climbs"`
	BikeSharingProvidersIds []int    `json:"bikeSharingProvidersIds"`
	DesiredLengthMeters     int      `json:"desiredLengthMeters"`
	AddRouteGeoJson         bool     `json:"addRouteGeoJson"`
	OptimizeWaypointOrder   bool     `json:"optimizeWaypointOrder"`
}


func initContext() *context.Context {
	context := &context.Context{}
	smtpServer := config.C().SMTP_Server
	smtpPort := config.C().SMTP_Port
	smtpUsername := config.C().SMTP_Username
	smtpToken := config.C().SMTP_Token
	// cyclersUrl := config.C().Cyclers_URL
	// cyclersKey := config.C().Cyclers_Api_Key

	receiverEmail := config.C().TEST_Receivers
	//This is the arbitrary max size of an email message
	//for a Zoleo device
	// maxEmailCharacterLimit := 200

	log.Printf("<<<<<<Start Debug information>>>>>: \n")
	log.Printf("SMTP Host: %s\n", smtpServer)
	log.Printf("SMTP Port: %s\n", smtpPort)
	log.Printf("SMTP Username: %s\n", smtpUsername)
	log.Printf("SMTP Password: %s\n", smtpToken)
	log.Printf("Receiver Email: %s\n", receiverEmail)
	
	log.Printf("Will be using the email address for sending schedules: '%s',\n", smtpUsername)
	log.Printf("Using '%s' database connection string", config.C().Db_Conn_Str)
	log.Printf("Using '%s' RabbitMQ connection string", config.C().Rabbit_Url)

	// test message 

	message := []byte("From: " + smtpUsername + "\r\n" +
					"To: " + receiverEmail + "\r\n" +
					"Subject: initContext in libretaxi.go\r\n" +
					"\r\n" +
					"This is a test email sent from a Go program.\r\n")

	auth := smtp.PlainAuth("", smtpUsername, smtpToken, smtpServer)

	// Sending email.
	
	if err := smtp.SendMail(smtpServer+":"+ strconv.FormatInt(smtpPort, 10), auth, smtpUsername, []string{receiverEmail}, message); err != nil {
		fmt.Println(err)
		return nil
	}
	fmt.Println("Email sent successfully")

	getEmails()

	db, err := sql.Open("postgres", config.C().Db_Conn_Str)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Print("Successfully connected to the database")
	}

	// context.Bot = bot
	context.Repo = repository.NewRepository(db)
	context.Config = config.C()
	return context
}



// ValidateZOLEOMessage validates an email message based on ZOLEO standards.
func ValidateZOLEOMessage(message string) error {
	// Check message length
	if len(message) > 200 {
		return errors.New("message exceeds 200 characters")
	}

	// Check for absence of signatures or historical messages
	// This is a basic check, might need more sophisticated logic
	if strings.Contains(message, "---") || strings.Contains(message, "From:") {
		return errors.New("message contains signatures or historical messages")
	}

	// Check for only basic text and emojis
	if !isBasicTextAndEmojis(message) {
		return errors.New("message contains more than basic text and emojis")
	}

	// Check for absence of attachments or HTML
	// Simple HTML tags check
	if strings.Contains(message, "<") && strings.Contains(message, ">") {
		return errors.New("HTML content detected in the message")
	}

	return nil
}

// isBasicTextAndEmojis checks if the string contains only basic text and emojis.
func isBasicTextAndEmojis(s string) bool {
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsPunct(r) || unicode.IsSpace(r) || isEmoji(r)) {
			return false
		}
	}
	return true
}

// isEmoji checks if the rune is an emoji.
// This is a simplistic check and may need more complex logic to handle all emoji cases.
func isEmoji(r rune) bool {
	return regexp.MustCompile(`[\x{1F600}-\x{1F64F}]`).MatchString(string(r))
}

// Message producer (app logic)
func main1() {
	context := initContext()
	log.Println("Starting message producer, rabbitmq, hoppity hop")
	context.RabbitPublish = rabbit.NewRabbitClient(config.C().Rabbit_Url, "messages")

	// test message 
	message := []byte("To: " + config.C().TEST_Receivers + "\r\n" +
					"Subject: Hello from Golang!\r\n" +
					"\r\n" +
					"This is a test email sent from a Go program.\r\n")

	auth := smtp.PlainAuth("", config.C().SMTP_Username, config.C().SMTP_Token, config.C().SMTP_Server)

	// Sending email.
	err := smtp.SendMail(config.C().SMTP_Username+":"+ strconv.FormatInt(config.C().SMTP_Port, 10), auth, config.C().SMTP_Username, []string{config.C().TEST_Receivers}, message)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Email sent successfully")

	// u := tgbotapi.NewUpdate(0)
	// u.Timeout = 60
	// u.Limit = 99

	// updates, _ := context.Bot.GetUpdatesChan(u)

	// for update := range updates {
	// 	if update.Message != nil {

	// 		// Ignore the case where message comes from a chat, not from user. We do not support chats.
	// 		if update.Message.From == nil {
	// 			continue
	// 		}

	// 		userId := update.Message.Chat.ID

	// 		log.Printf("[%d - %s] %s", userId, update.Message.From.UserName, update.Message.Text)
	// 		menu.HandleMessage(context, userId, update.Message)

	// 	} else if update.CallbackQuery != nil {

	// 		// A couple of places where we directly interact with Telegram API without a queue. Not a good thing,
	// 		// must be enqueued as well.

	// 		cb := update.CallbackQuery
	// 		context.Bot.AnswerCallbackQuery(tgbotapi.NewCallback(cb.ID, "👌 OK"))

	// 		emptyKeyboard := tgbotapi.NewInlineKeyboardMarkup([]tgbotapi.InlineKeyboardButton{})
	// 		removeButton := tgbotapi.NewEditMessageReplyMarkup(cb.Message.Chat.ID, cb.Message.MessageID, emptyKeyboard)

	// 		_, err := context.Bot.Send(removeButton)
	// 		if err != nil {
	// 			log.Println(err)
	// 		}


	// 		callback.NewTgCallbackHandler().Handle(context, cb.Data)
	// 	}
	// }
}

// Message consumer (send to Telegram respecting rate limits)
func main2() {
	context := initContext()
	context.RabbitConsume = rabbit.NewRabbitClient(config.C().Rabbit_Url, "messages")
	s := sender.NewSender(context)
	log.Println("Starting message consumer, rabbitmq, hoppity hop")
	s.Start()
}

func getLocale(languageCode string) *gotext.Locale {
	locale := gotext.NewLocale("./locales", "all")

	if languageCode == "ru" || languageCode == "es" {
		locale.AddDomain(languageCode)
	} else {
		locale.AddDomain("en")
	}
	return locale
}

// func massAnnounce() {
// 	ctx := &context.Context{}
// 	db, err := sql.Open("postgres", config.C().Db_Conn_Str)
// 	if err != nil {
// 		log.Fatal(err)
// 	} else {
// 		log.Print("Successfully connected to the database")
// 	}

// 	ctx.Repo = repository.NewRepository(db)
// 	ctx.Config = config.C()
// 	ctx.RabbitPublish = rabbit.NewRabbitClient(config.C().Rabbit_Url, "messages")

// 	var userId int64
// 	var languageCode string

// 	rows, err := db.Query("select \"userId\", \"languageCode\" from users where \"languageCode\"='pt-br'")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer rows.Close()

// 	rl := ratelimit.New(5) // don't load DB too much

// 	for rows.Next() {
// 		err := rows.Scan(&userId, &languageCode)
// 		if err == nil && ctx.Repo.ShowCallout(userId, "pt_br_translation_announcement") {

// 			// locale := getLocale(languageCode)
// 			// link := locale.Get("main.welcome_link")
// 			// text := link + " 👉👉👉 /start 👈👈👈"
// 			// msg := tgbotapi.NewMessage(userId, text)

// 			// ctx.RabbitPublish.PublishTgMessage(rabbit.MessageBag{
// 			// 	Message: msg,
// 			// 	Priority: 0, // LOWEST
// 			// })

// 			log.Println("Mass sending to ", userId, languageCode)

// 			ctx.Repo.DismissCallout(userId, "pt_br_translation_announcement")

// 			rl.Take()
// 		}
// 	}
// 	err = rows.Err()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }

func getEmails() {
	imapUsername := config.C().IMAP_Username
	imapHostname := config.C().IMAP_Hostname
	imapPort := config.C().IMAP_Port
	imapPassword := config.C().IMAP_Password
	imapFolder := config.C().IMAP_Folder
	imapCert := config.C().IMAP_Cert
	///Need to figure out where to put the SSL flag
	// imapSSL := config.C().IMAP_SSL 
	// Connect to the server
	log.Printf("Connecting to imap server")
	c, err := client.Dial(imapHostname + ":" + strconv.FormatInt(imapPort,10))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Using a certificate, from imap_cert %s", imapCert)
	caCert, err := ioutil.ReadFile(imapCert)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}
	log.Printf("Establishing TLS connection")
	if err := c.StartTLS(tlsConfig); err != nil {
		log.Fatal(err)
	}

	log.Printf("Logging in")
	// Authenticate
	if err := c.Login(imapUsername, imapPassword); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")

	log.Println("Listing mailboxes")
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func () {
		done <- c.List("", "*", mailboxes)
	}()
	
	log.Println("Mailboxes:")
	for m := range mailboxes {
		log.Println("* " + m.Name)
	}

	if err := <- done; err != nil {
		log.Fatal(err)
	}

	// Select INBOX
	// err := c.Select(imapFolder, false)
	//Use the mbox later for  retrieval of emails
	mbox, err := c.Select(imapFolder, false)

	if err != nil {
		log.Fatal(err)
	}

	if mbox.Messages == 0 {
		log.Println("No messages in mailbox")
		return
	}

	// Search for specific emails
	criteria := imap.NewSearchCriteria()
	criteria.Header.Add("From", "sender@example.com")
	ids, err := c.Search(criteria)
	if err != nil {
	log.Fatal(err)
	}

	if len(ids) == 0 {
		log.Println("No emails found")
		return
	}

	// Fetch specific emails
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(ids...)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 10)
	go func() {
		if err := c.Fetch(seqSet, items, messages); err != nil {
			log.Fatal(err)
		}
	}()

	for msg := range messages {
		r := msg.GetBody(section)
		if r == nil {
			log.Fatal("Server didn't return message body")
		}
	
		// Read and print the message body
		if body, err := ioutil.ReadAll(r); err == nil {
			fmt.Printf("Message Body:\n%s\n", string(body))
		} else {
			log.Fatal(err)
		}
	}
}

func test_payload() {
	apiURL := config.C().Cyclers_URL
	apiKey := config.C().Cyclers_Api_Key
	payload := RoutingRequest {
		Client: "IOS",
		Origin: Location { Point {Lat: 50.105827, Lon: 14.415478}},
		Destination: Location {Point {Lat: 50.105827, Lon: 14.415478}},
		Waypoints: []Location {
			
				{Point: Point {Lat: 50.081327, Lon: 14.413480}},
				
			// Add other waypoints as needed
			
			
		},
		Key: apiKey,
	}
	payload.Settings.OptimizeWaypointOrder = true
	jsonData, err := json.Marshal(payload)
	if err != nil {
		// handle error
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		// handle error
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// handle error
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
	}

	fmt.Println(string(body))
}

func main() {
	rand.Seed(time.Now().UnixNano())
	config.Init("libretaxi")

	go test_payload()
	// go main1()
	// go getEmails()
	/////// go massAnnounce()() ///has not been implemented

	forever := make(chan bool)
	<- forever
}
