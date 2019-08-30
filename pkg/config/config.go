package config

import (
	"io/ioutil"

	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Auth struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type NumPolicy struct {
	Num int `yaml:"number"`
}

type Policy struct {
	// Type of the policy, e.g. "number", "recentlyNotTouched"
	Type string `yaml:"type"`
	// RetainNum configure policy to retain given number tags in repo
	NumPolicy *NumPolicy `yaml:"numberPolicy,omitempty"`
	// RetainTags is tag patterns to be retained
	RetainTags []string `yaml:"retainTags"`
}

type C struct {
	Host     string   `yaml:"host"`
	Version  string   `yaml:"version"`
	Auth     Auth     `yaml:"auth"`
	Projects []string `yaml:"projects"`
	Policy   Policy   `yaml:"policy"`
}

var Config = C{}

func Load(configFile string) error {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		logrus.WithField("f", configFile).Error("Read config file error: ", err)
		return err
	}

	err = yaml.Unmarshal(b, &Config)
	if err != nil {
		logrus.WithField("f", configFile).Error("Unmarshal config file error: ", err)
		return err
	}

	if err = Normalize(&Config); err != nil {
		return err
	}

	return nil
}

func Normalize(c *C) error {
	trimed := strings.TrimSpace(c.Version)
	if len(trimed) < 3 {
		return fmt.Errorf("unrecoganized version %s, please provide version like 1.4, 1.7.5", c.Version)
	}

	c.Version = trimed[:3]
	return nil
}
