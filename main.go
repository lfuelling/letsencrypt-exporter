package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Config struct {
	LetsencryptPath string
	Domains         []string
	Port            int
}

var config Config

func loadCertificates() ([]tls.Certificate, error) {
	var certs []tls.Certificate

	for _, d := range config.Domains {
		log.Println("Loading certificate for '" + d + "'...")
		certPath := config.LetsencryptPath + "/" + d + "cert.pem"
		keyPath := config.LetsencryptPath + "/" + d + "privkey.pem"
		cer, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return certs, err
		}
		certs = append(certs, cer)
	}
	return certs, nil
}

func renderMetricsResponse() (string, error) {
	now := time.Now().Unix()
	certs, err := loadCertificates()
	if err != nil {
		return "", err
	}

	res := "# HELP letsencrypt_cert_not_after The certificate's NotAfter date as unix time'.\n" +
		"# TYPE letsencrypt_cert_not_after gauge"
	for _, crt := range certs {
		for _, dnsName := range crt.Leaf.DNSNames {
			res += `letsencrypt_cert_not_after{domain="` + dnsName + `"} ` + string(crt.Leaf.NotAfter.Unix()) + ` ` + string(now)
		}
	}
	return res, nil
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "/metrics" {
		response, err := renderMetricsResponse()
		if err != nil {
			log.Println("Error fetching metrics!", err)
			w.WriteHeader(500)
			return
		}
		_, _ = fmt.Fprint(w, response)
	} else {
		log.Println("Not found: '" + r.RequestURI)
		w.WriteHeader(404)
	}
}

func main() {
	file, ferr := os.Open("config.json")
	defer file.Close()
	if ferr != nil {
		log.Fatal("Unable to open config file!", ferr)
		return
	}
	decoder := json.NewDecoder(file)
	config = Config{}
	err := decoder.Decode(&config)
	if err != nil {
		log.Fatal("Unable to read config!", err)
		return
	}

	server := &http.Server{
		Addr:         ":" + string(config.Port),
		Handler:      http.HandlerFunc(handleMetrics),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
