package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"

	"github.com/lucas-clemente/quic-go"
)

const addr = "localhost:4242"
const message = "echo this"

type loggingWriter struct{ io.Writer }

func main() {
	listener, err := quic.ListenAddr(addr, generateTLSConfig(), nil)
	defer listener.Close()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			connections, err := listener.Accept(context.Background())
			if err != nil {
				log.Fatal(err)
			}

			stream, err := connections.AcceptStream(context.Background())
			if err != nil {
				log.Fatal(err)
			}

			_, err = io.Copy(loggingWriter{stream}, stream)
			if err != nil {
				log.Fatal(err)
			}
		}
	}()

	err = clientMain()
	if err != nil {
		log.Fatal(err)
	}
}

func clientMain() error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}

	conn, err := quic.DialAddr(addr, tlsConf, nil)
	if err != nil {
		return err
	}

	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		return err
	}

	fmt.Printf("Client: Sending '%s'\n", message)
	_, err = stream.Write([]byte(message))
	if err != nil {
		return err
	}

	buf := make([]byte, len(message))
	_, err = io.ReadFull(stream, buf)
	if err != nil {
		return err
	}
	fmt.Printf("Client: Got '%s'\n", buf)

	return nil
}

func (w loggingWriter) Write(b []byte) (int, error) {
	fmt.Printf("Server: Go '%s'\n", string(b))
	return w.Writer.Write(b)
}

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}

	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-example"},
	}
}
