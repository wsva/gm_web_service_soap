package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	wl_fs "github.com/wsva/lib_go/fs"
	wl_http "github.com/wsva/lib_go/http"
	mlib "github.com/wsva/monitor_lib_go"
	ml_detail "github.com/wsva/monitor_lib_go/detail"
)

type TargetWebService struct {
	Name             string `json:"Name"`
	Address          string `json:"Address"`
	URL              string `json:"URL"`
	ContentType      string `json:"ContentType"`
	SoapMessage      string `json:"SoapMessage"`
	StringInResponse string `json:"StringInResponse"`
}

var (
	MainTargetFile = "gm_web_service_soap_targets.json"
)

var targetList []TargetWebService

var resultsRuntime []mlib.MR
var resultsRuntimeLock sync.Mutex

func main() {
	err := initGlobals()
	if err != nil {
		fmt.Println(err)
		return
	}
	wg := &sync.WaitGroup{}
	for _, m := range targetList {
		go checkWebService(m, wg)
		wg.Add(1)
	}
	wg.Wait()

	jsonBytes, err := json.Marshal(resultsRuntime)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(mlib.MessageTypeMRList + string(jsonBytes))
}

func initGlobals() error {
	basepath, err := wl_fs.GetExecutableFullpath()
	if err != nil {
		return err
	}
	MainTargetFile = path.Join(basepath, MainTargetFile)

	contentBytes, err := os.ReadFile(MainTargetFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(contentBytes, &targetList)
}

func checkWebService(t TargetWebService, wg *sync.WaitGroup) {
	defer wg.Done()

	var md ml_detail.MDCommon
	defer func() {
		jsonString, err := md.JSONString()
		resultsRuntimeLock.Lock()
		if err != nil {
			resultsRuntime = append(resultsRuntime,
				mlib.GetMR(t.Name, t.Address, mlib.MTypeWebService, "", err.Error()))
		} else {
			resultsRuntime = append(resultsRuntime,
				mlib.GetMR(t.Name, t.Address, mlib.MTypeWebService, jsonString, ""))
		}
		resultsRuntimeLock.Unlock()
	}()

	httpclient := wl_http.HttpClient{
		Address:   t.URL,
		Method:    http.MethodPost,
		Data:      strings.NewReader(t.SoapMessage),
		Timeout:   10,
		HeaderMap: map[string]string{"Content-Type": t.ContentType},
	}
	bodyBytes, err := httpclient.DoRequest()
	if err != nil {
		md.Detail = err.Error()
	} else {
		if strings.Contains(string(bodyBytes), t.StringInResponse) {
			md.Detail = "ok"
		} else {
			md.Detail = "response incorrect"
		}
	}
}
