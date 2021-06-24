package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"webhook/image_blocker/handle"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	certFile string
	keyFile  string
	port     int
)

type Config struct {
	CertFile string
	KeyFile  string
}

func init() {
	certFile = "/etc/admission-controller/tls/tls.crt"
	keyFile = "/etc/admission-controller/tls/tls.key"
	port = 9527
}

func handler1(w http.ResponseWriter, r *http.Request) {
	serve(w, r)
}

func serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Fatalf("contentType=%s, expect application/json", contentType)
		return
	}

	log.Println(fmt.Sprintf("handling request: %s", body))

	admissionReview := v1.AdmissionReview{}

	err := json.Unmarshal(body, &admissionReview)
	if err != nil {
		msg := fmt.Sprintf("Request could not be decoded: %v", err)
		log.Fatalln(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	log.Println(admissionReview)
	admissionReview.Response = &v1.AdmissionResponse{
		Allowed: true,
		UID:     admissionReview.Request.UID,
	}
	image := []string{}
	pod := corev1.Pod{}
	if err := json.Unmarshal(admissionReview.Request.Object.Raw, &pod); err != nil {
		msg := fmt.Sprintf("Something went wrong while unmarshalling pod object: %+v", err)
		log.Fatalln(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	if !handle.MaditNS(pod.Namespace) {
		for _, container := range pod.Spec.Containers {
			if !handle.MaditImageList(container.Image) {
				image = append(image, container.Image)
				admissionReview.Response.Allowed = false
				admissionReview.Response.Result = &metav1.Status{
					Message: "using  Images is not allowed",
				}
				break
			}
		}
	}

	if admissionReview.Response.Allowed {
		log.Println("All images accepted")
	} else {
		log.Println("Rejected images: %v", image)
	}

	respBytes, err := json.Marshal(admissionReview)
	if err != nil {
		log.Fatalln(err)
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respBytes); err != nil {
		log.Fatalln(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func configTLS(config Config) *tls.Config {
	sCert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		log.Fatal(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{sCert},
	}
}

func main() {
	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
	config := Config{
		CertFile: certFile,
		KeyFile:  keyFile,
	}
	http.HandleFunc("/", handler1)
	//tr := &http.Transport{
	//	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	//}
	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", port),
		TLSConfig: configTLS(config),
	}
	err := server.ListenAndServeTLS("", "")
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Println("start success")
}
