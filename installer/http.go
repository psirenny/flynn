package installer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/flynn/flynn/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/flynn/flynn/Godeps/_workspace/src/github.com/badgerodon/ioutil"
	"github.com/flynn/flynn/Godeps/_workspace/src/github.com/julienschmidt/httprouter"
	log "github.com/flynn/flynn/Godeps/_workspace/src/gopkg.in/inconshreveable/log15.v2"
	"github.com/flynn/flynn/pkg/httphelper"
	"github.com/flynn/flynn/pkg/random"
	"github.com/flynn/flynn/pkg/sse"
)

type installerJSConfig struct {
	Endpoints map[string]string
}

type awsInputData struct {
	Creds        awsInputCreds `json:"creds"`
	Region       string        `json:"region"`
	InstanceType string        `json:"instance_type"`
	NumInstances int           `json:"num_instances"`
}

type awsInputCreds struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

type httpPrompt struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
	Yes     bool   `json:"yes,omitempty"`
	Input   string `json:"input,omitempty"`
}

type httpEvent struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Prompt      *httpPrompt `json:"prompt,omitempty"`
}

type HTTPInstallerStack struct {
	ID            string           `json:"id"`
	Stack         *Stack           `json:"-"`
	PromptOutChan chan *httpPrompt `json:"-"`
	PromptInChan  chan *httpPrompt `json:"-"`
	StreamMux     sync.Mutex       `json:"-"`
}

func (s *HTTPInstallerStack) YesNoPrompt(msg string) bool {
	fmt.Println("YesNoPrompt", msg, s.ID)
	s.PromptOutChan <- &httpPrompt{
		Type:    "yes_no",
		Message: msg,
	}
	fmt.Println("YesNoPrompt waiting...")
	res := <-s.PromptInChan
	fmt.Println("YesNoPrompt", res)
	return res.Yes
}

func (s *HTTPInstallerStack) PromptInput(msg string) string {
	fmt.Println("PromptInput", msg)
	s.PromptOutChan <- &httpPrompt{
		Type:    "input",
		Message: msg,
	}
	fmt.Println("PromptInput waiting...")
	res := <-s.PromptInChan
	fmt.Println("PromptInput", res)
	return res.Input
}

var httpInstallerStacks = make(map[string]*HTTPInstallerStack)
var httpInstallerStackMux sync.Mutex

func ServeHTTP() {
	httpRouter := httprouter.New()

	httpRouter.GET("/", serveTemplate)
	httpRouter.POST("/install", installHandler)
	httpRouter.GET("/events/:id", eventsHandler)
	httpRouter.POST("/prompt/:id", promptHandler)
	httpRouter.GET("/application.js", serveApplicationJS)
	httpRouter.GET("/assets/*assetPath", serveAsset)

	addr := ":4000"
	fmt.Printf("Navigate to http://localhost%s in your web browser to get started.\n", addr)
	http.ListenAndServe(addr, corsHandler(httpRouter))
}

func corsHandler(main http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httphelper.CORSAllowAllHandler(w, r)
		main.ServeHTTP(w, r)
	})
}

func installHandler(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	var input *awsInputData
	if err := httphelper.DecodeJSON(req, &input); err != nil {
		httphelper.Error(w, err)
		return
	}
	httpInstallerStackMux.Lock()
	defer httpInstallerStackMux.Unlock()
	var id = random.Hex(16)
	var creds aws.CredentialsProvider
	if input.Creds.AccessKeyID != "" && input.Creds.SecretAccessKey != "" {
		creds = aws.Creds(input.Creds.AccessKeyID, input.Creds.SecretAccessKey, "")
	} else {
		var err error
		creds, err = aws.EnvCreds()
		if err != nil {
			httphelper.Error(w, &httphelper.JSONError{
				Code:    httphelper.ValidationError,
				Message: err.Error(),
			})
			return
		}
	}
	s := &HTTPInstallerStack{
		ID:            id,
		PromptOutChan: make(chan *httpPrompt),
		PromptInChan:  make(chan *httpPrompt),
	}
	s.Stack = &Stack{
		Creds:        creds,
		Region:       input.Region,
		InstanceType: input.InstanceType,
		NumInstances: input.NumInstances,
		PromptInput:  s.PromptInput,
		YesNoPrompt:  s.YesNoPrompt,
	}
	if err := s.Stack.RunAWS(); err != nil {
		httphelper.Error(w, err)
		return
	}
	httpInstallerStacks[id] = s
	httphelper.JSON(w, 200, s)
}

func eventsHandler(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	id := params.ByName("id")
	s := httpInstallerStacks[id]
	if s == nil {
		httphelper.Error(w, &httphelper.JSONError{
			Code:    httphelper.NotFoundError,
			Message: "install instance not found",
		})
		return
	}

	// only allow a single client to connect at a time
	s.StreamMux.Lock()
	defer s.StreamMux.Unlock()

	eventChan := make(chan *httpEvent)

	l := log.New()
	stream := sse.NewStream(w, eventChan, l)
	stream.Serve()

	l.Info(fmt.Sprintf("streaming events for %s", s.ID))

	go func() {
		for {
			select {
			case event := <-s.Stack.EventChan:
				l.Info(event.Description)
				eventChan <- &httpEvent{
					Type:        "status",
					Description: event.Description,
				}
			case prompt := <-s.PromptOutChan:
				l.Info(prompt.Message)
				eventChan <- &httpEvent{
					Type:   "prompt",
					Prompt: prompt,
				}
			case err := <-s.Stack.ErrChan:
				l.Info(err.Error())
				stream.Error(err)
			case <-s.Stack.Done:
				l.Info("stack install complete")
				if s.Stack.Domain != nil {
					l.Info("sending domain")
					eventChan <- &httpEvent{
						Type:        "domain",
						Description: s.Stack.Domain.Name,
					}
				}
				if s.Stack.DashboardLoginToken != "" {
					l.Info("sending DashboardLoginToken")
					eventChan <- &httpEvent{
						Type:        "dashboard_login_token",
						Description: s.Stack.DashboardLoginToken,
					}
				}
				if s.Stack.CACert != "" {
					l.Info("sending CACert")
					eventChan <- &httpEvent{
						Type:        "ca_cert",
						Description: base64.URLEncoding.EncodeToString([]byte(s.Stack.CACert)),
					}
				}
				l.Info("closing stream")
				eventChan <- &httpEvent{
					Type: "done",
				}
				stream.Close()
				return
			}
		}
	}()

	stream.Wait()

	l.Info(s.Stack.DashboardLoginMsg())
}

func promptHandler(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	id := params.ByName("id")
	s := httpInstallerStacks[id]
	if s == nil {
		httphelper.Error(w, &httphelper.JSONError{
			Code:    httphelper.NotFoundError,
			Message: "install instance not found",
		})
		return
	}

	var input *httpPrompt
	if err := httphelper.DecodeJSON(req, &input); err != nil {
		httphelper.Error(w, err)
		return
	}
	s.PromptInChan <- input
	w.WriteHeader(200)
}

func serveApplicationJS(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	path := filepath.Join("app", "build", "application.js")
	f, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}

	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		fmt.Println(err)
		return
	}

	var jsConf bytes.Buffer
	jsConf.Write([]byte("window.InstallerConfig = "))
	json.NewEncoder(&jsConf).Encode(installerJSConfig{
		Endpoints: map[string]string{
			"install": "/install",
			"events":  "/events",
		},
	})
	jsConf.Write([]byte(";\n"))

	r := ioutil.NewMultiReadSeeker(bytes.NewReader(jsConf.Bytes()), f)

	http.ServeContent(w, req, path, fi.ModTime(), r)
}

func serveAsset(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	http.ServeFile(w, req, filepath.Join("app", "build", params.ByName("assetPath")))
}

func serveTemplate(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	http.ServeFile(w, req, filepath.Join("app", "installer.html"))
}
