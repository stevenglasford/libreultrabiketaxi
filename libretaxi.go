package main

import (
	"database/sql"
	// "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/leonelquinteros/gotext"
	_ "github.com/lib/pq" // important
	"go.uber.org/ratelimit"
	"libretaxi/callback"
	"libretaxi/config"
	"libretaxi/context"
	"libretaxi/menu"
	"libretaxi/rabbit"
	"libretaxi/repository"
	"libretaxi/sender"
	"log"
	"math/rand"
	"time"
	"net/smtp"
	"fmt"
)

func initContext() *context.Context {
	smtpServer := config.C().SMTP_Server
	smtpPort := config.C().SMTP_Port
	smtpUsername := config.C().SMTP_Username
	smtpToken := config.C().SMTP_Token
	receiverEmail := config.C().TEST_Receivers
	//This is the arbitrary max size of an email message
	//for a Zoleo device
	maxEmailCharacterLimit := 200

	log.Printf("<<<<<<Start Debug information>>>>>: \n")
	log.Printf("SMTP Host: %s\n", smtpHost)
	log.Printf("SMTP Port: %s\n", smtpPort)
	log.Printf("SMTP Username: %s\n", smtpUsername)
	log.Printf("SMTP Password: %s\n", smtpPassword)
	log.Printf("Receiver Email: %s\n", receiverEmail)
	
	log.Printf("Will be using the email address for sending schedules: '%s',\n", config.C.SMTP_Username)
	log.Printf("Using '%s' database connection string", config.C().Db_Conn_Str)
	log.Printf("Using '%s' RabbitMQ connection string", config.C().Rabbit_Url)


	context := &context.Context{}

	// bot, err := tgbotapi.NewBotAPI(config.C().Telegram_Token)
	// if err != nil {
	// 	log.Panic(err)
	// }
	// bot.Debug = true

	// log.Printf("Authorized on account %s", bot.Self.UserName)

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

// Message producer (app logic)
func main1() {
	context := initContext()
	context.RabbitPublish = rabbit.NewRabbitClient(config.C().Rabbit_Url, "messages")

	// test message 
	message := []byte("To: " + receiverEmail + "\r\n" +
					"Subject: Hello from Golang!\r\n" +
					"\r\n" +
					"This is a test email sent from a Go program.\r\n")

	auth := smtp.PlainAuth("", smtpUsername, smtpToken, smtpServer)

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, senderEmail, []string{receiverEmail}, message)
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

func massAnnounce() {
	context := &context.Context{}
	db, err := sql.Open("postgres", config.C().Db_Conn_Str)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Print("Successfully connected to the database")
	}

	context.Repo = repository.NewRepository(db)
	context.Config = config.C()
	context.RabbitPublish = rabbit.NewRabbitClient(config.C().Rabbit_Url, "messages")

	var userId int64
	var languageCode string

	rows, err := db.Query("select \"userId\", \"languageCode\" from users where \"languageCode\"='pt-br'")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	rl := ratelimit.New(5) // don't load DB too much

	for rows.Next() {
		err := rows.Scan(&userId, &languageCode)
		if err == nil && context.Repo.ShowCallout(userId, "pt_br_translation_announcement") {

			locale := getLocale(languageCode)
			link := locale.Get("main.welcome_link")
			text := link + " 👉👉👉 /start 👈👈👈"
			msg := tgbotapi.NewMessage(userId, text)

			context.RabbitPublish.PublishTgMessage(rabbit.MessageBag{
				Message: msg,
				Priority: 0, // LOWEST
			})

			log.Println("Mass sending to ", userId, languageCode)

			context.Repo.DismissCallout(userId, "pt_br_translation_announcement")

			rl.Take()
		}
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	config.Init("libretaxi")

	go main1()
	// go main2()
	//go massAnnounce()

	forever := make(chan bool)
	<- forever
}
