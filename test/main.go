package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"
)

type s struct {
	http.Server
}

var tunnelEstablishedResponseLine = []byte("HTTP/1.1 200 Connection established\r\n\r\n")

func (s *s) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, _ := os.OpenFile("log.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	defer f.Close()
	if r.RequestURI == "/cert" {
		data, _ := ioutil.ReadFile("./cert.pem")
		w.Header().Add("Content-Type", "application/x-x509-ca-cert")
		w.Header().Add("Content-Disposition", `attachment; filename="cert.pem"`)
		w.Write(data)
		return
	}
	if r.Method == http.MethodConnect {
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Write(tunnelEstablishedResponseLine)
		conf, err := generateTLSConfig(r.Host)
		if err != nil {
			log.Println(err)
			return
		}
		tlsConn := tls.Server(conn, conf)
		if err := tlsConn.Handshake(); err != nil {
			log.Println(err)
			return
		}
		buf := bufio.NewReader(tlsConn)
		request, err := http.ReadRequest(buf)
		if err != nil {
			log.Println(err)
			return
		}
		request.Host = r.Host
		request.URL.Scheme = "https"
		request.RemoteAddr = r.RemoteAddr
		request.RequestURI = ""
		request.URL.Host = r.Host

		resp, err := http.DefaultTransport.RoundTrip(request)
		defer resp.Body.Close()

		//resp, err := http.DefaultClient.Do(request)
		if err != nil {
			log.Println(err)
			return
		}
		mw := io.MultiWriter(tlsConn, f)
		resp.Write(mw)
	}
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate)
	server := new(s)
	http.ListenAndServe(":1111", server)
}

func generateTLSConfig(host string) (*tls.Config, error) {
	if h, _, _ := net.SplitHostPort(host); h != "" {
		host = h
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); err != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else {
		template.DNSNames = append(template.DNSNames, host)
	}

	data, _ := ioutil.ReadFile("./cert.pem")
	block, _ := pem.Decode(data)

	rootCa, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	data, _ = ioutil.ReadFile("./key.pem")
	block, _ = pem.Decode(data)
	rootKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, rootCa, &priv.PublicKey, rootKey)
	if err != nil {
		return nil, err
	}
	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	}
	serverCert := pem.EncodeToMemory(certBlock)

	keyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	}
	serverKey := pem.EncodeToMemory(keyBlock)

	conf, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, err
	}

	tlsConf := &tls.Config{Certificates: []tls.Certificate{conf}}

	return tlsConf, nil
}
