// Copyright Project Harbor Authors
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

package group

import (
	"strings"
	"time"

	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/common/utils"

	"github.com/goharbor/harbor/src/common/dao"
	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/common/utils/log"
)

// AddUserGroup - Add User Group
func AddUserGroup(userGroup models.UserGroup) (int, error) {
	o := dao.GetOrmer()

	sql := "insert into user_group (group_name, group_type, ldap_group_dn, creation_time, update_time) values (?, ?, ?, ?, ?) RETURNING id"
	var id int
	now := time.Now()

	err := o.Raw(sql, userGroup.GroupName, userGroup.GroupType, utils.TrimLower(userGroup.LdapGroupDN), now, now).QueryRow(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// QueryUserGroup - Query User Group
func QueryUserGroup(query models.UserGroup) ([]*models.UserGroup, error) {
	o := dao.GetOrmer()
	sql := `select id, group_name, group_type, ldap_group_dn from user_group where 1=1 `
	sqlParam := make([]interface{}, 1)
	groups := []*models.UserGroup{}
	if len(query.GroupName) != 0 {
		sql += ` and group_name like ? `
		sqlParam = append(sqlParam, `%`+dao.Escape(query.GroupName)+`%`)
	}

	if query.GroupType != 0 {
		sql += ` and group_type = ? `
		sqlParam = append(sqlParam, query.GroupType)
	}

	if len(query.LdapGroupDN) != 0 {
		sql += ` and ldap_group_dn = ? `
		sqlParam = append(sqlParam, utils.TrimLower(query.LdapGroupDN))
	}
	if query.ID != 0 {
		sql += ` and id = ? `
		sqlParam = append(sqlParam, query.ID)
	}
	_, err := o.Raw(sql, sqlParam).QueryRows(&groups)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

// GetUserGroup ...
func GetUserGroup(id int) (*models.UserGroup, error) {
	userGroup := models.UserGroup{ID: id}
	userGroupList, err := QueryUserGroup(userGroup)
	if err != nil {
		return nil, err
	}
	if len(userGroupList) > 0 {
		return userGroupList[0], nil
	}
	return nil, nil
}

// DeleteUserGroup ...
func DeleteUserGroup(id int) error {
	userGroup := models.UserGroup{ID: id}
	o := dao.GetOrmer()
	_, err := o.Delete(&userGroup)
	if err == nil {
		// Delete all related project members
		sql := `delete from project_member where entity_id = ? and entity_type='g'`
		_, err := o.Raw(sql, id).Exec()
		if err != nil {
			return err
		}
	}
	return err
}

// UpdateUserGroupName ...
func UpdateUserGroupName(id int, groupName string) error {
	log.Debugf("Updating user_group with id:%v, name:%v", id, groupName)
	o := dao.GetOrmer()
	sql := "update user_group set group_name = ? where id =  ? "
	_, err := o.Raw(sql, groupName, id).Exec()
	return err
}

// OnBoardUserGroup will check if a usergroup exists in usergroup table, if not insert the usergroup and
// put the id in the pointer of usergroup model, if it does exist, return the usergroup's profile.
// This is used for ldap and uaa authentication, such the usergroup can have an ID in Harbor.
// the keyAttribute and combinedKeyAttribute are key columns used to check duplicate usergroup in harbor
func OnBoardUserGroup(g *models.UserGroup, keyAttribute string, combinedKeyAttributes ...string) error {
	g.LdapGroupDN = utils.TrimLower(g.LdapGroupDN)

	o := dao.GetOrmer()
	created, ID, err := o.ReadOrCreate(g, keyAttribute, combinedKeyAttributes...)
	if err != nil {
		return err
	}

	if created {
		g.ID = int(ID)
	} else {
		prevGroup, err := GetUserGroup(int(ID))
		if err != nil {
			return err
		}
		g.ID = prevGroup.ID
		g.GroupName = prevGroup.GroupName
		g.GroupType = prevGroup.GroupType
		g.LdapGroupDN = prevGroup.LdapGroupDN
	}

	return nil
}

// GetGroupDNQueryCondition get the part of IN ('XXX', 'XXX') condition
func GetGroupDNQueryCondition(userGroupList []*models.UserGroup) string {
	result := make([]string, 0)
	count := 0
	for _, userGroup := range userGroupList {
		if userGroup.GroupType == common.LdapGroupType {
			result = append(result, "'"+userGroup.LdapGroupDN+"'")
			count++
		}
	}
	// No LDAP Group found
	if count == 0 {
		return ""
	}
	return strings.Join(result, ",")
}
