// Copyright 2018 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package config provide config for core api and other modules
// Before accessing user settings, need to call Load()
// For system settings, no need to call Load()
package config

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/goharbor/harbor/src/common"
	comcfg "github.com/goharbor/harbor/src/common/config"
	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/common/secret"
	"github.com/goharbor/harbor/src/common/utils/log"
	"github.com/goharbor/harbor/src/core/promgr"
	"github.com/goharbor/harbor/src/core/promgr/pmsdriver"
	"github.com/goharbor/harbor/src/core/promgr/pmsdriver/admiral"
	"github.com/goharbor/harbor/src/core/promgr/pmsdriver/local"
)

const (
	defaultKeyPath                     = "/etc/core/key"
	defaultTokenFilePath               = "/etc/core/token/tokens.properties"
	defaultRegistryTokenPrivateKeyPath = "/etc/core/private_key.pem"
)

var (
	// SecretStore manages secrets
	SecretStore *secret.Store
	// GlobalProjectMgr is initialized based on the deploy mode
	GlobalProjectMgr promgr.ProjectManager
	keyProvider      comcfg.KeyProvider
	// AdmiralClient is initialized only under integration deploy mode
	// and can be passed to project manager as a parameter
	AdmiralClient *http.Client
	// TokenReader is used in integration mode to read token
	TokenReader admiral.TokenReader
	// defined as a var for testing.
	defaultCACertPath = "/etc/core/ca/ca.crt"
	cfgMgr            *comcfg.CfgManager
)

// Init configurations
func Init() error {
	// init key provider
	initKeyProvider()

	cfgMgr = comcfg.NewDBCfgManager()

	log.Info("init secret store")
	// init secret store
	initSecretStore()
	log.Info("init project manager based on deploy mode")
	// init project manager based on deploy mode
	if err := initProjectManager(); err != nil {
		log.Errorf("Failed to initialise project manager, error: %v", err)
		return err
	}
	return nil
}

// InitWithSettings init config with predefined configs, and optionally overwrite the keyprovider
func InitWithSettings(cfgs map[string]interface{}, kp ...comcfg.KeyProvider) {
	Init()
	cfgMgr = comcfg.NewInMemoryManager()
	cfgMgr.UpdateConfig(cfgs)
	if len(kp) > 0 {
		keyProvider = kp[0]
	}
}

func initKeyProvider() {
	path := os.Getenv("KEY_PATH")
	if len(path) == 0 {
		path = defaultKeyPath
	}
	log.Infof("key path: %s", path)

	keyProvider = comcfg.NewFileKeyProvider(path)
}

func initSecretStore() {
	m := map[string]string{}
	m[JobserviceSecret()] = secret.JobserviceUser
	SecretStore = secret.NewStore(m)
}

func initProjectManager() error {
	var driver pmsdriver.PMSDriver
	if WithAdmiral() {
		log.Debugf("Initialising Admiral client with certificate: %s", defaultCACertPath)
		content, err := ioutil.ReadFile(defaultCACertPath)
		if err != nil {
			return err
		}
		pool := x509.NewCertPool()
		if ok := pool.AppendCertsFromPEM(content); !ok {
			return fmt.Errorf("failed to append cert content into cert worker")
		}
		AdmiralClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					RootCAs: pool,
				},
			},
		}

		// integration with admiral
		log.Info("initializing the project manager based on PMS...")
		path := os.Getenv("SERVICE_TOKEN_FILE_PATH")
		if len(path) == 0 {
			path = defaultTokenFilePath
		}
		log.Infof("service token file path: %s", path)
		TokenReader = &admiral.FileTokenReader{
			Path: path,
		}
		driver = admiral.NewDriver(AdmiralClient, AdmiralEndpoint(), TokenReader)
	} else {
		// standalone
		log.Info("initializing the project manager based on local database...")
		driver = local.NewDriver()
	}
	GlobalProjectMgr = promgr.NewDefaultProjectManager(driver, true)
	return nil

}

// GetCfgManager return the current config manager
func GetCfgManager() *comcfg.CfgManager {
	if cfgMgr == nil {
		return comcfg.NewDBCfgManager()
	}
	return cfgMgr
}

// Load configurations
func Load() error {
	return cfgMgr.Load()
}

// Upload save all system configurations
func Upload(cfg map[string]interface{}) error {
	return cfgMgr.UpdateConfig(cfg)
}

// GetSystemCfg returns the system configurations
func GetSystemCfg() (map[string]interface{}, error) {
	sysCfg := cfgMgr.GetAll()
	if len(sysCfg) == 0 {
		return nil, errors.New("can not load system config, the database might be down")
	}
	return sysCfg, nil
}

// AuthMode ...
func AuthMode() (string, error) {
	err := cfgMgr.Load()
	if err != nil {
		log.Errorf("failed to load config, error %v", err)
		return "db_auth", err
	}
	return cfgMgr.Get(common.AUTHMode).GetString(), nil
}

// TokenPrivateKeyPath returns the path to the key for signing token for registry
func TokenPrivateKeyPath() string {
	path := os.Getenv("TOKEN_PRIVATE_KEY_PATH")
	if len(path) == 0 {
		path = defaultRegistryTokenPrivateKeyPath
	}
	return path
}

// LDAPConf returns the setting of ldap server
func LDAPConf() (*models.LdapConf, error) {
	err := cfgMgr.Load()
	if err != nil {
		return nil, err
	}
	return &models.LdapConf{
		LdapURL:               cfgMgr.Get(common.LDAPURL).GetString(),
		LdapSearchDn:          cfgMgr.Get(common.LDAPSearchDN).GetString(),
		LdapSearchPassword:    cfgMgr.Get(common.LDAPSearchPwd).GetString(),
		LdapBaseDn:            cfgMgr.Get(common.LDAPBaseDN).GetString(),
		LdapUID:               cfgMgr.Get(common.LDAPUID).GetString(),
		LdapFilter:            cfgMgr.Get(common.LDAPFilter).GetString(),
		LdapScope:             cfgMgr.Get(common.LDAPScope).GetInt(),
		LdapConnectionTimeout: cfgMgr.Get(common.LDAPTimeout).GetInt(),
		LdapVerifyCert:        cfgMgr.Get(common.LDAPVerifyCert).GetBool(),
	}, nil
}

// LDAPGroupConf returns the setting of ldap group search
func LDAPGroupConf() (*models.LdapGroupConf, error) {
	err := cfgMgr.Load()
	if err != nil {
		return nil, err
	}
	return &models.LdapGroupConf{
		LdapGroupBaseDN:              cfgMgr.Get(common.LDAPGroupBaseDN).GetString(),
		LdapGroupFilter:              cfgMgr.Get(common.LDAPGroupSearchFilter).GetString(),
		LdapGroupNameAttribute:       cfgMgr.Get(common.LDAPGroupAttributeName).GetString(),
		LdapGroupSearchScope:         cfgMgr.Get(common.LDAPGroupSearchScope).GetInt(),
		LdapGroupAdminDN:             cfgMgr.Get(common.LdapGroupAdminDn).GetString(),
		LdapGroupMembershipAttribute: cfgMgr.Get(common.LDAPGroupMembershipAttribute).GetString(),
	}, nil
}

// TokenExpiration returns the token expiration time (in minute)
func TokenExpiration() (int, error) {
	return cfgMgr.Get(common.TokenExpiration).GetInt(), nil
}

// RobotTokenDuration returns the token expiration time of robot account (in minute)
func RobotTokenDuration() int {
	return cfgMgr.Get(common.RobotTokenDuration).GetInt()
}

// ExtEndpoint returns the external URL of Harbor: protocol://host:port
func ExtEndpoint() (string, error) {
	return cfgMgr.Get(common.ExtEndpoint).GetString(), nil
}

// ExtURL returns the external URL: host:port
func ExtURL() (string, error) {
	endpoint, err := ExtEndpoint()
	if err != nil {
		log.Errorf("failed to load config, error %v", err)
	}
	l := strings.Split(endpoint, "://")
	if len(l) > 0 {
		return l[1], nil
	}
	return endpoint, nil
}

// SecretKey returns the secret key to encrypt the password of target
func SecretKey() (string, error) {
	return keyProvider.Get(nil)
}

// SelfRegistration returns the enablement of self registration
func SelfRegistration() (bool, error) {
	return cfgMgr.Get(common.SelfRegistration).GetBool(), nil
}

// RegistryURL ...
func RegistryURL() (string, error) {
	return cfgMgr.Get(common.RegistryURL).GetString(), nil
}

// InternalJobServiceURL returns jobservice URL for internal communication between Harbor containers
func InternalJobServiceURL() string {
	return strings.TrimSuffix(cfgMgr.Get(common.JobServiceURL).GetString(), "/")
}

// InternalCoreURL returns the local harbor core url
func InternalCoreURL() string {
	return strings.TrimSuffix(cfgMgr.Get(common.CoreURL).GetString(), "/")

}

// InternalTokenServiceEndpoint returns token service endpoint for internal communication between Harbor containers
func InternalTokenServiceEndpoint() string {
	return InternalCoreURL() + "/service/token"
}

// InternalNotaryEndpoint returns notary server endpoint for internal communication between Harbor containers
// This is currently a conventional value and can be unaccessible when Harbor is not deployed with Notary.
func InternalNotaryEndpoint() string {
	return cfgMgr.Get(common.NotaryURL).GetString()
}

// InitialAdminPassword returns the initial password for administrator
func InitialAdminPassword() (string, error) {
	return cfgMgr.Get(common.AdminInitialPassword).GetString(), nil
}

// OnlyAdminCreateProject returns the flag to restrict that only sys admin can create project
func OnlyAdminCreateProject() (bool, error) {
	return cfgMgr.Get(common.ProjectCreationRestriction).GetString() == common.ProCrtRestrAdmOnly, nil
}

// Email returns email server settings
func Email() (*models.Email, error) {
	err := cfgMgr.Load()
	if err != nil {
		return nil, err
	}
	return &models.Email{
		Host:     cfgMgr.Get(common.EmailHost).GetString(),
		Port:     cfgMgr.Get(common.EmailPort).GetInt(),
		Username: cfgMgr.Get(common.EmailUsername).GetString(),
		Password: cfgMgr.Get(common.EmailPassword).GetString(),
		SSL:      cfgMgr.Get(common.EmailSSL).GetBool(),
		From:     cfgMgr.Get(common.EmailFrom).GetString(),
		Identity: cfgMgr.Get(common.EmailIdentity).GetString(),
		Insecure: cfgMgr.Get(common.EmailInsecure).GetBool(),
	}, nil
}

// Database returns database settings
func Database() (*models.Database, error) {
	database := &models.Database{}
	database.Type = cfgMgr.Get(common.DatabaseType).GetString()
	postgresql := &models.PostGreSQL{
		Host:     cfgMgr.Get(common.PostGreSQLHOST).GetString(),
		Port:     cfgMgr.Get(common.PostGreSQLPort).GetInt(),
		Username: cfgMgr.Get(common.PostGreSQLUsername).GetString(),
		Password: cfgMgr.Get(common.PostGreSQLPassword).GetString(),
		Database: cfgMgr.Get(common.PostGreSQLDatabase).GetString(),
		SSLMode:  cfgMgr.Get(common.PostGreSQLSSLMode).GetString(),
	}
	database.PostGreSQL = postgresql

	return database, nil
}

// CoreSecret returns a secret to mark harbor-core when communicate with
// other component
func CoreSecret() string {
	return os.Getenv("CORE_SECRET")
}

// JobserviceSecret returns a secret to mark Jobservice when communicate with
// other component
// TODO replace it with method of SecretStore
func JobserviceSecret() string {
	return os.Getenv("JOBSERVICE_SECRET")
}

// WithNotary returns a bool value to indicate if Harbor's deployed with Notary
func WithNotary() bool {
	return cfgMgr.Get(common.WithNotary).GetBool()
}

// WithClair returns a bool value to indicate if Harbor's deployed with Clair
func WithClair() bool {
	return cfgMgr.Get(common.WithClair).GetBool()
}

// ClairEndpoint returns the end point of clair instance, by default it's the one deployed within Harbor.
func ClairEndpoint() string {
	return cfgMgr.Get(common.ClairURL).GetString()
}

// ClairDB return Clair db info
func ClairDB() (*models.PostGreSQL, error) {
	clairDB := &models.PostGreSQL{
		Host:     cfgMgr.Get(common.ClairDBHost).GetString(),
		Port:     cfgMgr.Get(common.ClairDBPort).GetInt(),
		Username: cfgMgr.Get(common.ClairDBUsername).GetString(),
		Password: cfgMgr.Get(common.ClairDBPassword).GetString(),
		Database: cfgMgr.Get(common.ClairDB).GetString(),
		SSLMode:  cfgMgr.Get(common.ClairDBSSLMode).GetString(),
	}
	return clairDB, nil
}

// AdmiralEndpoint returns the URL of admiral, if Harbor is not deployed with admiral it should return an empty string.
func AdmiralEndpoint() string {
	if cfgMgr.Get(common.AdmiralEndpoint).GetString() == "NA" {
		return ""
	}
	return cfgMgr.Get(common.AdmiralEndpoint).GetString()
}

// ScanAllPolicy returns the policy which controls the scan all.
func ScanAllPolicy() models.ScanAllPolicy {
	var res models.ScanAllPolicy
	log.Infof("Scan all policy %v", cfgMgr.Get(common.ScanAllPolicy).GetString())
	if err := json.Unmarshal([]byte(cfgMgr.Get(common.ScanAllPolicy).GetString()), &res); err != nil {
		log.Errorf("Failed to unmarshal the value in configuration for Scan All policy, error: %v, returning the default policy", err)
		return models.DefaultScanAllPolicy
	}
	return res
}

// WithAdmiral returns a bool to indicate if Harbor's deployed with admiral.
func WithAdmiral() bool {
	return len(AdmiralEndpoint()) > 0
}

// UAASettings returns the UAASettings to access UAA service.
func UAASettings() (*models.UAASettings, error) {
	err := cfgMgr.Load()
	if err != nil {
		return nil, err
	}
	us := &models.UAASettings{
		Endpoint:     cfgMgr.Get(common.UAAEndpoint).GetString(),
		ClientID:     cfgMgr.Get(common.UAAClientID).GetString(),
		ClientSecret: cfgMgr.Get(common.UAAClientSecret).GetString(),
		VerifyCert:   cfgMgr.Get(common.UAAVerifyCert).GetBool(),
	}
	return us, nil
}

// ReadOnly returns a bool to indicates if Harbor is in read only mode.
func ReadOnly() bool {
	return cfgMgr.Get(common.ReadOnly).GetBool()
}

// WithChartMuseum returns a bool to indicate if chartmuseum is deployed with Harbor.
func WithChartMuseum() bool {
	return cfgMgr.Get(common.WithChartMuseum).GetBool()
}

// GetChartMuseumEndpoint returns the endpoint of the chartmuseum service
// otherwise an non nil error is returned
func GetChartMuseumEndpoint() (string, error) {
	chartEndpoint := strings.TrimSpace(cfgMgr.Get(common.ChartRepoURL).GetString())
	if len(chartEndpoint) == 0 {
		return "", errors.New("empty chartmuseum endpoint")
	}
	return chartEndpoint, nil
}

// GetRedisOfRegURL returns the URL of Redis used by registry
func GetRedisOfRegURL() string {
	return os.Getenv("_REDIS_URL_REG")
}

// GetPortalURL returns the URL of portal
func GetPortalURL() string {
	url := os.Getenv("PORTAL_URL")
	if len(url) == 0 {
		return common.DefaultPortalURL
	}
	return url
}

// GetRegistryCtlURL returns the URL of registryctl
func GetRegistryCtlURL() string {
	url := os.Getenv("REGISTRYCTL_URL")
	if len(url) == 0 {
		return common.DefaultRegistryCtlURL
	}
	return url
}

// GetClairHealthCheckServerURL returns the URL of
// the health check server of Clair
func GetClairHealthCheckServerURL() string {
	url := os.Getenv("CLAIR_HEALTH_CHECK_SERVER_URL")
	if len(url) == 0 {
		return common.DefaultClairHealthCheckServerURL
	}
	return url
}

// HTTPAuthProxySetting returns the setting of HTTP Auth proxy.  the settings are only meaningful when the auth_mode is
// set to http_auth
func HTTPAuthProxySetting() (*models.HTTPAuthProxy, error) {
	if err := cfgMgr.Load(); err != nil {
		return nil, err
	}
	return &models.HTTPAuthProxy{
		Endpoint:            cfgMgr.Get(common.HTTPAuthProxyEndpoint).GetString(),
		TokenReviewEndpoint: cfgMgr.Get(common.HTTPAuthProxyTokenReviewEndpoint).GetString(),
		VerifyCert:          cfgMgr.Get(common.HTTPAuthProxyVerifyCert).GetBool(),
		AlwaysOnBoard:       cfgMgr.Get(common.HTTPAuthProxyAlwaysOnboard).GetBool(),
	}, nil

}

// OIDCSetting returns the setting of OIDC provider, currently there's only one OIDC provider allowed for Harbor and it's
// only effective when auth_mode is set to oidc_auth
func OIDCSetting() (*models.OIDCSetting, error) {
	if err := cfgMgr.Load(); err != nil {
		return nil, err
	}
	scopeStr := cfgMgr.Get(common.OIDCScope).GetString()
	extEndpoint := strings.TrimSuffix(cfgMgr.Get(common.ExtEndpoint).GetString(), "/")
	scope := []string{}
	for _, s := range strings.Split(scopeStr, ",") {
		scope = append(scope, strings.TrimSpace(s))
	}

	return &models.OIDCSetting{
		Name:         cfgMgr.Get(common.OIDCName).GetString(),
		Endpoint:     cfgMgr.Get(common.OIDCEndpoint).GetString(),
		VerifyCert:   cfgMgr.Get(common.OIDCVerifyCert).GetBool(),
		ClientID:     cfgMgr.Get(common.OIDCCLientID).GetString(),
		ClientSecret: cfgMgr.Get(common.OIDCClientSecret).GetString(),
		RedirectURL:  extEndpoint + common.OIDCCallbackPath,
		Scope:        scope,
	}, nil
}
