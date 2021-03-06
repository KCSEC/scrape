package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var conf Config

func cleanKeys() {
	now := time.Now()

	for key, _ := range conf.keys {
		if now.Sub(conf.keys[key]) > conf.maxTime {
			delete(conf.keys, key)
		}
	}
}

func get(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[-] Could not access %s.\n", url)
		return []byte("")
	}

	if resp.StatusCode != 200 {
		log.Printf("[-] Received HTTP error %d.\n", resp.StatusCode)
		return []byte("")
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[-] Could not read response from %s.\n", url)
		return []byte("")
	}

	return body
}

func scrape() {
	var pastes []*Paste

	log.Println("[+] Checking for new pastes.")

	resp := get("https://pastebin.com/api_scraping.php?limit=100")
	err := json.Unmarshal(resp, &pastes)
	if err != nil {
		log.Println("[-] Could not parse list of pastes.")
		log.Printf("[-] %s.\n", err.Error())
		log.Println(string(resp))
		return
	}

	for i, _ := range pastes {
		p := pastes[i]
		p.Download()
		p.Process()
	}
}

func main() {
	conf = newConfig()
	for {
		scrape()
		time.Sleep(conf.sleep)
		cleanKeys()
	}
}
