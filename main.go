package main

import (
  "bufio"
  "encoding/json"
  "fmt"
  "io/ioutil"
  "net/http"
  "os"
  "reflect"
  "regexp"
  "strconv"
  "strings"
  "sync"
  "time"

  "github.com/PuerkitoBio/goquery"
  "github.com/Syfaro/telegram-bot-api"
)

var mutex sync.Mutex

// Config holds all application configuration parsed from environment variables.
type Config struct {
  Token            string
  DataFolder       string
  NotifyToChat     int64
  CheckingInterval int
  BatchSize        int
  UserAgent        string
}

var cfg Config

// loadConfig reads configuration from environment variables and sets defaults.
func loadConfig() Config {
  c := Config{
    Token:            os.Getenv("TOKEN"),
    DataFolder:       os.Getenv("DATA_FOLDER"),
    CheckingInterval: 300,
    BatchSize:        20,
    UserAgent:        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
  }

  if c.Token == "" {
    panic("TOKEN environment variable is required")
  }
  if c.DataFolder == "" {
    panic("DATA_FOLDER environment variable is required")
  }

  if v := os.Getenv("CHECKING_INTERVAL"); v != "" {
    parsed, err := strconv.Atoi(v)
    if err != nil {
      panic("CHECKING_INTERVAL must be a number: " + err.Error())
    }
    c.CheckingInterval = parsed
  }

  if v := os.Getenv("NOTIFY_TO_CHAT"); v != "" {
    parsed, err := strconv.ParseInt(v, 10, 64)
    if err != nil {
      panic("NOTIFY_TO_CHAT must be a number: " + err.Error())
    }
    c.NotifyToChat = parsed
  }

  if v := os.Getenv("BATCH_SIZE"); v != "" {
    parsed, err := strconv.Atoi(v)
    if err != nil {
      panic("BATCH_SIZE must be a number: " + err.Error())
    }
    c.BatchSize = parsed
  }

  if v := os.Getenv("USER_AGENT"); v != "" {
    c.UserAgent = v
  }

  return c
}

// Ad holds details extracted from an advertisement card.
type Ad struct {
  Link     string
  Title    string
  Price    string
  Features string
  Place    string
}

// collapseWhitespace replaces all runs of whitespace (including newlines) with a single space and trims edges.
func collapseWhitespace(s string) string {
  fields := strings.Fields(s)
  return strings.Join(fields, " ")
}

// readSeenAds reads sended_links.json as a map of link -> price.
// Returns an empty map if the file does not exist.
func readSeenAds(path string) (map[string]string, error) {
  data, err := os.ReadFile(path)
  if err != nil {
    if os.IsNotExist(err) {
      return make(map[string]string), nil
    }
    return nil, err
  }
  result := make(map[string]string)
  if err := json.Unmarshal(data, &result); err != nil {
    return make(map[string]string), nil
  }
  return result, nil
}

// writeSeenAds writes the link -> price map as JSON.
func writeSeenAds(ads map[string]string, path string) error {
  data, err := json.Marshal(ads)
  if err != nil {
    return err
  }
  return os.WriteFile(path, data, 0644)
}

func _check(err error) {
  if err != nil {
    panic(err)
  }
}

func createFile(path string) {
  // check if file exists
  var _, err = os.Stat(path)

  // create file if not exists
  if os.IsNotExist(err) {
    var file, err = os.Create(path)
    _check(err)

    defer file.Close()
  }

  fmt.Println("File Created Successfully", path)
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
  file, err := os.Open(path)
  if err != nil {
    return nil, err
  }
  defer file.Close()

  var lines []string
  scanner := bufio.NewScanner(file)
  for scanner.Scan() {
    lines = append(lines, scanner.Text())
  }
  return lines, scanner.Err()
}

// writeLines writes the lines to the given file.
func writeLines(lines []string, path string) error {
  file, err := os.Create(path)
  if err != nil {
    return err
  }
  defer file.Close()

  w := bufio.NewWriter(file)
  for _, line := range lines {
    fmt.Fprintln(w, line)
  }
  return w.Flush()
}

func Contains(a []string, x string) bool {
  for _, n := range a {
    if x == n {
      return true
    }
  }
  return false
}

const urlSample = "Correct URL sample: https://www.bazaraki.com/real-estate/houses-and-villas-rent/lemesos-district-limassol/?price_min=500&price_max=1000"

// validateAndFetchURL fetches the given URL and validates that it is a valid
// Bazaraki listing page. Returns the parsed document or a descriptive error.
func validateAndFetchURL(url string, client *http.Client) (*goquery.Document, error) {
  if client == nil {
    client = http.DefaultClient
  }

  req, err := http.NewRequest("GET", url, nil)
  if err != nil {
    return nil, fmt.Errorf("Invalid URL: %w", err)
  }
  req.Header.Set("User-Agent", cfg.UserAgent)

  res, err := client.Do(req)
  if err != nil {
    return nil, fmt.Errorf("Failed to fetch URL: %w", err)
  }
  defer res.Body.Close()

  if res.StatusCode != 200 {
    return nil, fmt.Errorf("Server returned status %d for the URL", res.StatusCode)
  }

  doc, err := goquery.NewDocumentFromReader(res.Body)
  if err != nil {
    return nil, fmt.Errorf("Failed to parse page content: %w", err)
  }

  if len(doc.Find(".list-announcement-assortiments").Nodes) == 0 {
    return nil, fmt.Errorf("No advertisements found on the page. Make sure the URL points to a Bazaraki listing page.")
  }

  return doc, nil
}

func telegramBot() {
  bot, err := tgbotapi.NewBotAPI(cfg.Token)
  _check(err)

  u := tgbotapi.NewUpdate(0)

  updates, err := bot.GetUpdatesChan(u)

  for update := range updates {
    if update.Message == nil {
      continue
    }

    data_folder := cfg.DataFolder + "/"

    // Make sure that message in text
    if reflect.TypeOf(update.Message.Text).Kind() == reflect.String && update.Message.Text != "" {

      chat_folder := data_folder + strconv.FormatInt(update.Message.Chat.ID, 10)

      switch update.Message.Text {
      case "/start":

        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hi, i'm a Bazaraki notification bot!")
        bot.Send(msg)
        msg1 := tgbotapi.NewMessage(update.Message.Chat.ID, "Send me Bazaraki advertisements list URL sorted by newest to start receiving notifications.")
        bot.Send(msg1)
        msg2 := tgbotapi.NewMessage(update.Message.Chat.ID, "To stop receiving notifications send me /stop")
        bot.Send(msg2)

        if cfg.NotifyToChat != 0 {
          msg := tgbotapi.NewMessage(cfg.NotifyToChat, "New user: @" + update.Message.From.UserName)
          bot.Send(msg)
        }

        fmt.Println("Start chat with id:" + strconv.FormatInt(update.Message.Chat.ID, 10) + ". User: @" + update.Message.From.UserName)

      case "/stop":
        err := os.RemoveAll(chat_folder)
        _check(err)

        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "You successfully stop following all advertisements")
        bot.Send(msg)

      default:

        url := update.Message.Text

        fmt.Println("request: " + url)

        _, err := validateAndFetchURL(url, nil)
        if err != nil {
          bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
          bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, urlSample))
          continue
        }

        // Create advertisement list
        os.MkdirAll(chat_folder, os.ModePerm);

        advertisements_path := chat_folder + "/advertisements"
        os.Remove(advertisements_path)
        createFile(advertisements_path)

        writeSeenAds(make(map[string]string), chat_folder + "/sended_links.json")

        // Write url to advertisements list
        lines, err := readLines(advertisements_path)
        _check(err)

        lines = append(lines, url)

        err = writeLines(lines, advertisements_path)
        _check(err)

        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Now you are following only this url: " + url)
        bot.Send(msg)
        check_updates(true)
      }
    } else {
      msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Send URL for subscribe")
      bot.Send(msg)

    }
  }
}

// formatAdEntry formats a single Ad into a text block.
func formatAdEntry(ad Ad) string {
  var entry string
  if ad.Title != "" {
    entry += ad.Title + "\n"
  }
  if ad.Price != "" {
    entry += ad.Price + "\n"
  }
  if ad.Features != "" {
    entry += ad.Features + "\n"
  }
  if ad.Place != "" {
    entry += ad.Place + "\n"
  }
  entry += ad.Link
  return entry
}

// sendAdBatch sends ads in batches to a Telegram chat.
// If header is non-empty, it is prepended to the first batch message.
func sendAdBatch(bot *tgbotapi.BotAPI, chatID int64, ads []Ad, header string) {
  batchSize := cfg.BatchSize
  for i := 0; i < len(ads); i += batchSize {
    end := i + batchSize
    if end > len(ads) {
      end = len(ads)
    }
    batch := ads[i:end]
    var text string
    if header != "" && i == 0 {
      text = header + "\n\n"
    }
    for j, ad := range batch {
      entry := formatAdEntry(ad)
      if len(ads) == 1 && header == "" {
        text += entry
      } else {
        text += fmt.Sprintf("%d. %s", i+j+1, entry)
      }
      if j < len(batch)-1 {
        text += "\n\n"
      }
    }
    bot.Send(tgbotapi.NewMessage(chatID, text))
  }
}

func check_updates(notify bool) {
  mutex.Lock()
  defer mutex.Unlock()

  bot, err := tgbotapi.NewBotAPI(cfg.Token)
  _check(err)

  data_folder := cfg.DataFolder + "/"

  folders, err := ioutil.ReadDir(data_folder)
  _check(err)


  for _, folder := range folders {
    if folder.IsDir() {
      chat_id := folder.Name()

      advertisements, err := readLines(data_folder + chat_id + "/advertisements")
      _check(err)

      for _, url := range advertisements {
        seenAdsPath := data_folder + chat_id + "/sended_links.json"

        seenAds, err := readSeenAds(seenAdsPath)
        _check(err)

        doc, err := validateAndFetchURL(url, nil)
        if err != nil {
          fmt.Println("Error fetching URL: " + url + " - " + err.Error())
          continue
        }

        advertisements_container := doc.Find(".list-simple__output")

        // Remove ads from other regions
        other_advertisments_header_index := advertisements_container.Find("h2.header").First().Index()

        advertisements := advertisements_container.Children()
        if other_advertisments_header_index != -1 {
          advertisements = advertisements_container.Children().Slice(0, other_advertisments_header_index)
        }

        var newAds []Ad
        var priceChanges []Ad
        advertisements.Find("a").Each(func(i int, s *goquery.Selection) {
          link, _ := s.Attr("href")
          isAdv, _ := regexp.MatchString(`/adv/\d{7}_.*/`, link)
          relevantAd := !s.HasClass("js-advert-gallery-item") && s.HasClass("mask")

          if isAdv && relevantAd {
            advert := s.Closest(".advert")
            title := collapseWhitespace(advert.Find(".advert__content-title").Text())
            price := collapseWhitespace(advert.Find(".advert__content-price").Text())
            features := collapseWhitespace(advert.Find(".advert__content-features").Text())
            place := collapseWhitespace(advert.Find(".advert__content-place").Text())

            oldPrice, seen := seenAds[link]
            if !seen {
              seenAds[link] = price
              if notify {
                newAds = append(newAds, Ad{
                  Link:     "https://www.bazaraki.com" + link,
                  Title:    title,
                  Price:    price,
                  Features: features,
                  Place:    place,
                })
              }
            } else if oldPrice != "" && oldPrice != price {
              seenAds[link] = price
              if notify {
                priceChanges = append(priceChanges, Ad{
                  Link:     "https://www.bazaraki.com" + link,
                  Title:    title,
                  Price:    oldPrice + " → " + price,
                  Features: features,
                  Place:    place,
                })
              }
            }
          }
        })

        // Send new ads in batches
        if notify && len(newAds) > 0 {
          chat_id_int, err := strconv.ParseInt(chat_id, 10, 64)
          _check(err)
          sendAdBatch(bot, chat_id_int, newAds, "")
          fmt.Printf("Sent %d new ads to chat %s\n", len(newAds), chat_id)
        }

        // Send price change notifications
        if notify && len(priceChanges) > 0 {
          chat_id_int, err := strconv.ParseInt(chat_id, 10, 64)
          _check(err)
          sendAdBatch(bot, chat_id_int, priceChanges, "Price updates:")
          fmt.Printf("Sent %d price changes to chat %s\n", len(priceChanges), chat_id)
        }

        err = writeSeenAds(seenAds, seenAdsPath)
        _check(err)
      }
    }
  }
}

func main() {
  cfg = loadConfig()

  fmt.Printf("Starting with checking interval: %ds, data folder: %s\n", cfg.CheckingInterval, cfg.DataFolder)

  go telegramBot()

  for {
    check_updates(true)
    time.Sleep(time.Second * time.Duration(cfg.CheckingInterval))
  }
}

