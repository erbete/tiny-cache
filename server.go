package tinycache

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"tinycache/proto"

	"github.com/lucas-clemente/quic-go"
	"google.golang.org/grpc"
)

// https://github.com/sssgun/grpc-quic
type server struct {
	proto.UnimplementedCacheServer
}

func (s *server) Add(ctx context.Context, in *proto.AddRequest) (*proto.AddResponse, error) {
	log.Print("TODO: Add")
	return nil, nil
}

func (s *server) Get(ctx context.Context, in *proto.GetRequest) (*proto.GetResponse, error) {
	log.Print("TODO: Get")
	return nil, nil
}

func Run(address string) error {
	listener, err := quic.ListenAddr(address, generateTLSConfig(), nil)
	defer listener.Close()
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	proto.RegisterCacheServer(grpcServer, &server{})

	for {
		connection, err := listener.Accept(context.Background())
		if err != nil {
			return err
		}

		go func(conn quic.Connection) error {
			stream, err := connection.AcceptStream(context.Background())
			defer stream.Close()
			if err != nil {
				return err
			}

			buffer := make([]byte, 1024)
			_, err = stream.Read(buffer)
			if err != nil {
				return err
			}

			fmt.Printf("@Server: '%s'\n", string(buffer))

			responseMsg := "hi from server"
			_, err = stream.Write([]byte(responseMsg)) // respond back to client
			if err != nil {
				return err
			}

			return nil
		}(connection)
	}
}

// func generateTLSConfig(certFile, keyFile string) (*tls.Config, error) {
// 	if len(certFile) > 0 && len(keyFile) > 0 {
// 		log.Printf("generateTLSConfig] certFile=%s, keyFile=%s", certFile, keyFile)
//
// 		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
// 		if err != nil {
// 			log.Printf("failed to tls.LoadX509KeyPair. %s", err.Error())
// 			return nil, err
// 		}
// 		return &tls.Config{
// 			Certificates: []tls.Certificate{cert},
// 			NextProtos:   []string{"quic-echo-example"},
// 		}, nil
// 	} else {
// 		log.Printf("generateTLSConfig] GenerateKey")
// 		key, err := rsa.GenerateKey(rand.Reader, 1024)
// 		if err != nil {
// 			log.Printf("failed to rsa.GenerateKey. %s", err.Error())
// 			return nil, err
// 		}
// 		template := x509.Certificate{SerialNumber: big.NewInt(1)}
// 		certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
// 		if err != nil {
// 			log.Printf("failed to x509.CreateCertificate. %s", err.Error())
// 			return nil, err
// 		}
// 		keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
// 		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
//
// 		tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
// 		if err != nil {
// 			log.Printf("failed to tls.X509KeyPair. %s", err.Error())
// 			return nil, err
// 		}
//
// 		return &tls.Config{
// 			Certificates: []tls.Certificate{tlsCert},
// 			NextProtos:   []string{"quic-echo-example"},
// 		}, nil
// 	}
// }

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
