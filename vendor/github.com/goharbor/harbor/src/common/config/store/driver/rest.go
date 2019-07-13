package driver

import (
	"errors"
	"github.com/goharbor/harbor/src/common/http"
	"github.com/goharbor/harbor/src/common/http/modifier"
	"github.com/goharbor/harbor/src/common/utils/log"
)

// RESTDriver - config store driver based on REST API
type RESTDriver struct {
	configRESTURL string
	client        *http.Client
}

// NewRESTDriver - Create RESTDriver
func NewRESTDriver(configRESTURL string, modifiers ...modifier.Modifier) *RESTDriver {
	return &RESTDriver{configRESTURL: configRESTURL, client: http.NewClient(nil, modifiers...)}
}

// Load - load config data from REST server
func (h *RESTDriver) Load() (map[string]interface{}, error) {
	cfgMap := map[string]interface{}{}
	log.Infof("get configuration from url: %+v", h.configRESTURL)
	err := h.client.Get(h.configRESTURL, &cfgMap)
	if err != nil {
		log.Errorf("Failed on load rest config err:%v, url:%v", err, h.configRESTURL)
	}
	if len(cfgMap) < 1 {
		return cfgMap, errors.New("failed to load rest config")
	}
	return cfgMap, err
}

// Save - Save config data to REST server by PUT method
func (h *RESTDriver) Save(cfgMap map[string]interface{}) error {
	return h.client.Put(h.configRESTURL, cfgMap)
}
