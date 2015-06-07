package agent

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func (this *Agent) createDockerApiHandler(dir string, endpoint string) http.Handler {
	var h http.Handler

	switch {
	case strings.Contains(endpoint, "http"):
		h = createTcpHandler(endpoint)
	case strings.Contains(endpoint, "unix"):
		h = createUnixHandler(endpoint)
	case strings.Contains(endpoint, "tcp"):
		h = createTlsHandler(endpoint, this.DockerSettings.Cert, this.DockerSettings.Key, this.DockerSettings.Ca)
	}
	return h
}

type UnixHandler struct {
	path string
}

func (h *UnixHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := net.Dial("unix", h.path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	c := httputil.NewClientConn(conn, nil)
	defer c.Close()

	res, err := c.Do(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer res.Body.Close()

	copyHeader(w.Header(), res.Header)
	if _, err := io.Copy(w, res.Body); err != nil {
		log.Println(err)
	}
}

type TlsHandler struct {
	path string
	Cert string
	Key  string
	Ca   string
}

func (h *TlsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if h.Cert == "" {
		panic(ErrNoDockerTlsCert)
	}
	if h.Key == "" {
		panic(ErrNoDockerTlsKey)
	}
	tlsCert, err := tls.LoadX509KeyPair(h.Cert, h.Key)
	if err != nil {
		panic(err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{tlsCert}}
	if h.Ca == "" {
		tlsConfig.InsecureSkipVerify = true
	} else {
		cert, err := ioutil.ReadFile(h.Ca)
		if err != nil {
			panic(err)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(cert) {
			panic(ErrBadDockerTlsCert)
		}
		tlsConfig.RootCAs = caPool
	}

	conn, err := tls.Dial("tcp", h.path, tlsConfig)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	c := httputil.NewClientConn(conn, nil)
	defer c.Close()

	res, err := c.Do(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer res.Body.Close()

	copyHeader(w.Header(), res.Header)
	if _, err := io.Copy(w, res.Body); err != nil {
		log.Println(err)
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func createTcpHandler(e string) http.Handler {
	u, err := url.Parse(e)
	if err != nil {
		log.Fatal(err)
	}
	return httputil.NewSingleHostReverseProxy(u)
}

func createUnixHandler(e string) http.Handler {
	ep := strings.Split(e, "unix://")[1]
	if _, err := os.Stat(ep); err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("unix socket %s does not exist", ep)
		}
		log.Fatal(err)
	}
	return &UnixHandler{ep}
}

func createTlsHandler(e string, cert, key, ca string) http.Handler {
	return &TlsHandler{path: strings.Split(e, "tcp://")[1], Cert: cert, Key: key, Ca: ca}
}
