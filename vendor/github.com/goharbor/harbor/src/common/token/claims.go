package token

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/goharbor/harbor/src/common/rbac"
)

// RobotClaims implements the interface of jwt.Claims
type RobotClaims struct {
	jwt.StandardClaims
	TokenID   int64          `json:"id"`
	ProjectID int64          `json:"pid"`
	Access    []*rbac.Policy `json:"access"`
}

// Valid valid the claims "tokenID, projectID and access".
func (rc RobotClaims) Valid() error {
	if rc.TokenID < 0 {
		return errors.New("Token id must an valid INT")
	}
	if rc.ProjectID < 0 {
		return errors.New("Project id must an valid INT")
	}
	if rc.Access == nil {
		return errors.New("The access info cannot be nil")
	}
	stdErr := rc.StandardClaims.Valid()
	if stdErr != nil {
		return stdErr
	}
	return nil
}
