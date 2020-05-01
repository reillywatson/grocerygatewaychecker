package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var (
	phone      = flag.String("phone", "", "Phone number to send to (ex +12349115678)")
	postalCode = flag.String("postal", "", "Postal code to check (ex m5v3v9)")
)

func main() {
	flag.Parse()
	if *phone == "" || *postalCode == "" {
		flag.Usage()
		os.Exit(1)
	}
	slots := map[string]bool{}
	for {
		avail, err := getAvailableSlots(*postalCode)
		if err != nil {
			fmt.Println("Error getting slots:", err)
		}
		oldSlots := slots
		slots = map[string]bool{}
		var newSlots []string
		for _, slot := range avail {
			slots[slot] = true
			if !oldSlots[slot] {
				newSlots = append(newSlots, slot)
			}
		}
		fmt.Printf("Found %d new slots, %d total slots\n", len(newSlots), len(slots))
		if len(newSlots) > 0 {
			fmt.Println("New slots:", newSlots)
			err = sendSms(*phone, fmt.Sprintf("Check out these new slots: %v", newSlots))
			if err != nil {
				fmt.Println("Error sending SMS:", err)
			}
		}
		time.Sleep(time.Second * 30)
	}
}

func getAvailableSlots(postalCode string) ([]string, error) {
	req, err := http.NewRequest("GET", "https://www.grocerygateway.com/store/groceryGateway/en/pre-select", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", fmt.Sprintf(`groceryGateway-postalCode="RES,%s"`, postalCode))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	var slots []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			isWindow := false
			date := ""
			time := ""
			available := false
			for _, a := range n.Attr {
				switch a.Key {
				case "data-deliverytitle":
					isWindow = true
				case "data-datekey":
					date = a.Val
				case "data-info":
					time = a.Val
				case "data-status":
					available = (a.Val != "BLOCKED")
				}
			}
			if isWindow && available {
				slots = append(slots, fmt.Sprintf("%s %s", date, time))
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return slots, nil
}

func sendSms(phone, msg string) error {
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	from := os.Getenv("TWILIO_FROM_NUMBER")
	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + accountSid + "/Messages.json"

	// Build out the data for our message
	v := url.Values{}
	v.Set("To", phone)
	v.Set("From", from)
	v.Set("Body", msg)
	rb := *strings.NewReader(v.Encode())

	// Create client
	client := &http.Client{}

	req, _ := http.NewRequest("POST", urlStr, &rb)
	req.SetBasicAuth(accountSid, authToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	// Make request
	_, err := client.Do(req)
	return err
}
